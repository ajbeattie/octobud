// Copyright (C) 2025 Austin Beattie
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

// Package main provides the main entry point for the worker.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"
	"go.uber.org/zap"
	"golang.org/x/term"

	config "github.com/ajbeattie/octobud/backend/internal/config"
	"github.com/ajbeattie/octobud/backend/internal/core/notification"
	"github.com/ajbeattie/octobud/backend/internal/core/pullrequest"
	"github.com/ajbeattie/octobud/backend/internal/core/repository"
	"github.com/ajbeattie/octobud/backend/internal/core/syncstate"
	"github.com/ajbeattie/octobud/backend/internal/db"
	"github.com/ajbeattie/octobud/backend/internal/github"
	"github.com/ajbeattie/octobud/backend/internal/jobs"
	"github.com/ajbeattie/octobud/backend/internal/sync"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	promptToken := flag.Bool(
		"prompt-token",
		false,
		"prompt for GitHub token instead of using GH_TOKEN env var",
	)
	flag.Parse()

	cfg := config.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-stop
		log.Println("worker: shutdown signal received")
		cancel()
	}()

	// Create pgx pool for River
	dbPool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		//nolint:gocritic // exitAfterDefer: intentional - fatal error, cleanup not needed
		log.Fatalf("worker: failed to create database pool: %v", err)
	}
	defer dbPool.Close()

	if pingErr := dbPool.Ping(ctx); pingErr != nil {
		log.Fatalf("worker: database ping failed: %v", pingErr)
	}

	// Create database/sql connection for existing sync service
	dbConn, dbErr := sql.Open("pgx", cfg.DatabaseURL)
	if dbErr != nil {
		log.Fatalf("worker: failed to open database connection: %v", dbErr)
	}
	defer func() {
		if err := dbConn.Close(); err != nil {
			log.Printf("worker: failed to close database connection: %v", err)
		}
	}()

	// Initialize GitHub client
	githubClient := github.NewClient()

	if *promptToken {
		log.Println("worker: prompting for GitHub token")
		token, tokenErr := promptGitHubToken()
		if tokenErr != nil {
			log.Fatalf("worker: failed to prompt for token: %v", tokenErr)
		}
		if setErr := githubClient.SetToken(ctx, token); setErr != nil {
			log.Fatalf("worker: failed to set prompted token: %v", setErr)
		}
		log.Println("worker: GitHub token configured via prompt")
	} else {
		if cfg.GHToken == "" {
			fmt.Fprintln(os.Stderr, "Error: GH_TOKEN environment variable is not set")
			fmt.Fprintln(os.Stderr, "Please set GH_TOKEN or use --prompt-token flag to enter it interactively")
			os.Exit(1)
		}
		log.Println("worker: using GitHub token from GH_TOKEN environment variable")
		if setErr := githubClient.SetToken(ctx, cfg.GHToken); setErr != nil {
			log.Fatalf("worker: failed to set GitHub token: %v", setErr)
		}
		log.Println("worker: GitHub token validated successfully")
	}

	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("worker: failed to create logger: %v", err)
	}

	// Initialize business logic services
	queries := db.New(dbConn)
	syncStateSvc := syncstate.NewSyncStateService(queries)
	repositorySvc := repository.NewService(queries)
	pullRequestSvc := pullrequest.NewService(queries)
	notificationSvc := notification.NewService(queries)

	// Initialize sync service
	syncService := sync.NewService(
		logger,
		time.Now,
		githubClient,
		syncStateSvc,
		repositorySvc,
		pullRequestSvc,
		notificationSvc,
		queries, // userStore for sync settings
	)

	// Configure periodic jobs
	var periodicJobs []*river.PeriodicJob
	syncInterval := cfg.SyncInterval
	if syncInterval == 0 {
		syncInterval = 20 * time.Second // Default to 20 seconds if not configured
	}

	// Periodic sync of notifications
	periodicJobs = append(periodicJobs, river.NewPeriodicJob(
		river.PeriodicInterval(syncInterval),
		func() (river.JobArgs, *river.InsertOpts) {
			return jobs.SyncNotificationsArgs{},

				&river.InsertOpts{
					Queue: "sync_notifications",
					UniqueOpts: river.UniqueOpts{
						ByState: []rivertype.JobState{
							rivertype.JobStateAvailable,
							rivertype.JobStatePending,
							rivertype.JobStateRunning,
							rivertype.JobStateRetryable,
							rivertype.JobStateScheduled,
						},
					},
				}
		},
		&river.PeriodicJobOpts{RunOnStart: true},
	))

	// Register workers (needs to be done before creating River client)
	log.Println("worker: registering River workers...")
	workers := river.NewWorkers()
	// We'll add the workers after creating the client since SyncNotificationsWorker needs it

	// Create River client
	riverClient, err := river.NewClient(riverpgxv5.New(dbPool), &river.Config{
		Queues: map[string]river.QueueConfig{
			"sync_notifications":   {MaxWorkers: 1},
			"process_notification": {MaxWorkers: 10}, // Allow parallel processing of notifications
			"apply_rule":           {MaxWorkers: 10},
		},
		Workers:      workers,
		PeriodicJobs: periodicJobs,
	})
	if err != nil {
		log.Fatalf("worker: failed to create River client: %v", err)
	}

	// Register workers after River client is created
	river.AddWorker(workers, jobs.NewSyncNotificationsWorker(logger, syncService, riverClient))
	river.AddWorker(
		workers,
		jobs.NewSyncOlderNotificationsWorker(logger, syncService, riverClient),
	)
	river.AddWorker(workers, jobs.NewProcessNotificationWorker(dbConn, syncService))
	river.AddWorker(workers, jobs.NewApplyRuleWorker(queries))
	log.Println(
		"worker: registered 4 workers (SyncNotifications, SyncOlderNotifications, ProcessNotification, ApplyRule)",
	)

	// Start River client
	log.Printf("worker: starting River client with sync interval: %s", syncInterval)
	if err := riverClient.Start(ctx); err != nil {
		log.Fatalf("worker: failed to start River client: %v", err)
	}

	// Wait for shutdown signal
	<-ctx.Done()

	log.Println("worker: shutting down River client")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := riverClient.Stop(shutdownCtx); err != nil {
		log.Printf("worker: graceful shutdown failed: %v", err)
	}

	log.Println("worker: stopped")
}

// promptGitHubToken prompts the user to enter their GitHub Personal Access Token.
// Returns the token string or an error if prompting fails.
func promptGitHubToken() (string, error) {
	// Set up signal handling for Ctrl-C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	// Create a channel to receive the token input
	tokenChan := make(chan string, 1)
	errChan := make(chan error, 1)

	go func() {
		fmt.Println("Please enter your GitHub Personal Access Token (PAT):")
		fmt.Println("The token needs 'notifications' and 'repo' scopes.")
		fmt.Print("> ")

		// Read password without echoing to terminal
		tokenBytes, err := term.ReadPassword(syscall.Stdin)
		fmt.Println() // Print newline after hidden input

		if err != nil {
			errChan <- fmt.Errorf("failed to read token: %w", err)
			return
		}

		tokenChan <- string(tokenBytes)
	}()

	// Wait for either token input or Ctrl-C
	select {
	case <-sigChan:
		fmt.Println("\nInterrupted")
		return "", fmt.Errorf("canceled by user")
	case err := <-errChan:
		return "", err
	case token := <-tokenChan:
		token = strings.TrimSpace(token)
		if token == "" {
			return "", fmt.Errorf("token cannot be empty")
		}
		return token, nil
	}
}
