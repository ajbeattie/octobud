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

// Package main provides the main entry point for the goose migrations.
package main

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		args = []string{"up"}
	}

	cmd := args[0]
	cmdArgs := args[1:]

	ctx := context.Background()
	databaseURL := os.Getenv("DATABASE_URL")

	// Run goose migrations using stdlib driver for compatibility
	poolConfig := mustParseConfig(databaseURL)
	db := stdlib.OpenDBFromPool(mustCreatePool(ctx, poolConfig))
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("failed to close database connection: %v", err)
		}
	}()

	if err := goose.SetDialect("postgres"); err != nil {
		//nolint:gocritic // exitAfterDefer: intentional - fatal error, cleanup not needed
		log.Fatalf("failed to set dialect: %v", err)
	}

	log.Println("Running application migrations...")
	if err := goose.RunContext(ctx, cmd, db, "/app/migrations", cmdArgs...); err != nil {
		log.Fatalf("failed to run goose migrations: %v", err)
	}
	log.Println("Application migrations completed successfully")

	// Run River migrations using pgxpool with riverpgxv5 driver
	if cmd == "up" {
		log.Println("Running River migrations...")
		pool, err := pgxpool.New(ctx, databaseURL)
		if err != nil {
			log.Fatalf("failed to create pgx pool for River migrations: %v", err)
		}
		defer pool.Close()

		migrator, err := rivermigrate.New(riverpgxv5.New(pool), nil)
		if err != nil {
			log.Fatalf("failed to create River migrator: %v", err)
		}
		_, err = migrator.Migrate(ctx, rivermigrate.DirectionUp, &rivermigrate.MigrateOpts{})
		if err != nil {
			log.Fatalf("failed to run River migrations: %v", err)
		}
		log.Println("River migrations completed successfully")
	}
}

func mustParseConfig(databaseURL string) *pgxpool.Config {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		log.Fatalf("failed to parse database URL: %v", err)
	}
	return config
}

func mustCreatePool(ctx context.Context, config *pgxpool.Config) *pgxpool.Pool {
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		log.Fatalf("failed to create pool: %v", err)
	}
	return pool
}
