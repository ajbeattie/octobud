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

// Package main provides the main entry point for the server.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"go.uber.org/zap"
	"golang.org/x/term"

	"github.com/ajbeattie/octobud/backend/internal/api"
	"github.com/ajbeattie/octobud/backend/internal/api/auth"
	"github.com/ajbeattie/octobud/backend/internal/api/shared"
	apiuser "github.com/ajbeattie/octobud/backend/internal/api/user"
	config "github.com/ajbeattie/octobud/backend/internal/config"
	authsvc "github.com/ajbeattie/octobud/backend/internal/core/auth"
	"github.com/ajbeattie/octobud/backend/internal/core/syncstate"
	store "github.com/ajbeattie/octobud/backend/internal/db"
	"github.com/ajbeattie/octobud/backend/internal/github"

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

	ctx := context.Background()

	// Create database/sql connection for queries
	dbConn, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("server: open database: %v", err)
	}
	defer func() {
		if err := dbConn.Close(); err != nil {
			log.Printf("server: failed to close database connection: %v", err)
		}
	}()

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if pingErr := dbConn.PingContext(pingCtx); pingErr != nil {
		//nolint:gocritic // exitAfterDefer: intentional - fatal error, cleanup not needed
		log.Fatalf("server: connect database: %v", pingErr)
	}

	queries := store.New(dbConn)

	// Initialize auth service and default user
	authService := authsvc.NewService(queries)
	if initErr := authService.InitializeDefaultUser(ctx); initErr != nil {
		log.Fatalf("server: failed to initialize default user: %v", initErr)
	}
	log.Println("server: default user initialized")

	// Create pgx pool for River client
	dbPool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("server: failed to create database pool: %v", err)
	}
	defer dbPool.Close()

	// Create River client for queueing jobs (but don't start workers - worker process handles that)
	riverClient, err := river.NewClient(riverpgxv5.New(dbPool), &river.Config{})
	if err != nil {
		log.Fatalf("server: failed to create River client: %v", err)
	}
	log.Println("server: River client initialized for job queueing")

	// Initialize GitHub client
	githubClient := github.NewClient()
	var apiHandler *api.Handler
	tokenConfigured := false

	// Try prompt first if requested
	if *promptToken {
		log.Println("server: prompting for GitHub token")
		token, promptErr := promptGitHubToken()
		if promptErr != nil {
			log.Printf("server: prompt failed: %v", promptErr)
			// Fall through to try GH_TOKEN if available
		} else {
			if setErr := githubClient.SetToken(ctx, token); setErr != nil {
				log.Printf("server: failed to set prompted token: %v", setErr)
				// Fall through to try GH_TOKEN if available
			} else {
				tokenConfigured = true
				log.Println("server: GitHub token configured via prompt")
			}
		}
	}

	// If no token configured yet, try GH_TOKEN environment variable
	if !tokenConfigured && cfg.GHToken != "" {
		log.Println("server: using GitHub token from GH_TOKEN environment variable")
		if setErr := githubClient.SetToken(ctx, cfg.GHToken); setErr != nil {
			log.Fatalf("server: failed to set GitHub token: %v", setErr)
		}
		tokenConfigured = true
		log.Println("server: GitHub token validated successfully")
	}

	// Initialize zap logger
	logger, err := zap.NewProduction()
	if err != nil {
		logger = zap.NewNop()
	}

	// Configure handler based on whether token is available
	if tokenConfigured {
		apiHandler = api.NewHandler(queries,
			api.WithSyncService(dbConn, githubClient, logger),
			api.WithRiverClient(riverClient))
		log.Println("server: GitHub client configured, refresh endpoint available")
	} else {
		fmt.Fprintln(os.Stderr, "Warning: No GitHub token configured")
		fmt.Fprintln(os.Stderr, "The refresh endpoint will be unavailable.")
		fmt.Fprintln(os.Stderr, "Set GH_TOKEN environment variable or use --prompt-token flag to enable it.")
		log.Println("server: no GitHub token configured, refresh endpoint will be unavailable")
		apiHandler = api.NewHandler(queries, api.WithRiverClient(riverClient))
	}

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	// Add security headers to all responses
	router.Use(shared.SecurityHeadersMiddleware)

	// Limit request body size to prevent DoS attacks (1MB default)
	router.Use(shared.BodyLimitMiddleware(shared.DefaultMaxBodySize))

	// Configure CORS
	corsConfig := configureCORS()
	router.Use(corsConfig.Handler)

	router.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte("ok"))
	})

	// Initialize rate limiter (5 attempts per minute per username)
	rateLimiter := auth.NewRateLimiter(5, 1*time.Minute, logger)

	// Initialize token revocation
	tokenRevocation := auth.NewTokenRevocation(logger)

	// Initialize sync state service
	syncStateSvc := syncstate.NewSyncStateService(queries)

	// Force secure cookies if environment variable is set
	// If not set, cookies will auto-detect HTTPS from request headers
	// (useful when behind reverse proxy that terminates TLS)
	secureCookies := os.Getenv("SECURE_COOKIES") == "true"

	userHandler := apiuser.New(
		logger,
		authService,
		cfg.JWTSecret,
		cfg.JWTExpiry,
		rateLimiter,
		tokenRevocation,
		secureCookies,
	).WithRiverClient(riverClient).WithSyncStateService(syncStateSvc)

	// Register API routes with auth middleware
	router.Route("/api", func(r chi.Router) {
		// Public user routes (login) - with rate limiting
		r.Group(func(r chi.Router) {
			r.Use(auth.RateLimitMiddleware(rateLimiter, logger))
			r.Post("/user/login", userHandler.HandleLogin)
		})

		// Protected user routes
		r.Group(func(r chi.Router) {
			r.Use(auth.JWTMiddleware(cfg.JWTSecret, logger, tokenRevocation))
			r.Use(auth.CSRFMiddleware(logger))
			r.Get("/user/me", userHandler.HandleGetCurrentUser)
			r.Post("/user/refresh", userHandler.HandleRefreshToken)
			r.Post("/user/logout", userHandler.HandleLogout)
			r.Put("/user/credentials", userHandler.HandleUpdateCredentials)
			r.Get("/user/sync-settings", userHandler.HandleGetSyncSettings)
			r.Put("/user/sync-settings", userHandler.HandleUpdateSyncSettings)
			r.Get("/user/sync-state", userHandler.HandleGetSyncState)
			r.Post("/user/sync-older", userHandler.HandleSyncOlder)
		})

		// All other API routes require auth and CSRF
		r.Group(func(r chi.Router) {
			r.Use(auth.JWTMiddleware(cfg.JWTSecret, logger, tokenRevocation))
			r.Use(auth.CSRFMiddleware(logger))
			apiHandler.Register(r)
		})
	})

	server := &http.Server{
		Addr:         cfg.Addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("server: listening on %s", cfg.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: listen error: %v", err)
		}
	}()

	waitForShutdown(server)
}

func waitForShutdown(server *http.Server) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	<-stop
	log.Println("server: shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("server: graceful shutdown failed: %v", err)
	}

	log.Println("server: stopped")
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

// configureCORS sets up CORS middleware with appropriate defaults
func configureCORS() *cors.Cors {
	// Get allowed origins from environment, default to empty (same-origin only)
	allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	var origins []string

	if allowedOrigins != "" {
		// Parse comma-separated list of origins
		for _, origin := range strings.Split(allowedOrigins, ",") {
			origin = strings.TrimSpace(origin)
			if origin != "" {
				origins = append(origins, origin)
			}
		}
	} else {
		// Default: allow same-origin and localhost for development
		origins = []string{
			"http://localhost:5173", // Frontend dev server
			"http://localhost:3000", // Frontend production
			"http://localhost:8080", // Backend (for testing)
		}
	}

	return cors.New(cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, // 5 minutes
	})
}
