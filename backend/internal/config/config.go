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

// Package config provides configuration for the application.
package config

import (
	"log"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config is the configuration for the application.
type Config struct {
	Addr         string
	DatabaseURL  string
	SyncInterval time.Duration
	GHToken      string
	JWTSecret    string
	JWTExpiry    time.Duration
}

// Load loads the configuration from the environment variables.
func Load() Config {
	// Load .env files with priority: root .env (../.env) overrides local .env
	// This allows us to use a single .env file at project root for both Docker and local development
	// Try local .env first (for backward compatibility or Docker containers)
	if err := godotenv.Load(".env"); err != nil && !os.IsNotExist(err) {
		log.Printf("config: error loading .env file: %v", err)
	}
	// Then try root .env (takes precedence if it exists)
	// When running from backend/ directory, this loads from project root
	if err := godotenv.Load("../.env"); err != nil && !os.IsNotExist(err) {
		log.Printf("config: error loading ../.env file: %v", err)
	}

	jwtSecret := getEnv("JWT_SECRET", "")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}

	// JWT expiration: default to 7 days with automatic refresh for active users
	// Active users stay logged in via automatic refresh, inactive users re-authenticate
	// Can be overridden with JWT_EXPIRY env var (e.g., "168h" for 7 days, "720h" for 30 days)
	jwtExpiry := getDurationEnv("JWT_EXPIRY")
	if jwtExpiry == 0 {
		jwtExpiry = 7 * 24 * time.Hour // 7 days default - balances security and UX
	}

	cfg := Config{
		Addr: getEnv("SERVER_ADDR", ":8080"),
		DatabaseURL: getEnv(
			"DATABASE_URL",
			"postgres://postgres:postgres@localhost:5432/octobud?sslmode=disable",
		),
		SyncInterval: getDurationEnv("SYNC_INTERVAL"),
		GHToken:      getEnv("GH_TOKEN", ""),
		JWTSecret:    jwtSecret,
		JWTExpiry:    jwtExpiry,
	}

	// Warn about default credentials
	warnAboutDefaultCredentials(cfg.DatabaseURL, cfg.Addr)

	return cfg
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}

	if fallback == "" {
		log.Printf("config: %s not set and no fallback provided", key)
	}

	return fallback
}

func getDurationEnv(key string) time.Duration {
	value, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(value) == "" {
		return 0
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		log.Printf("config: invalid duration for %s=%q: %v", key, value, err)
		return 0
	}

	if duration <= 0 {
		log.Printf("config: non-positive duration for %s=%q", key, value)
		return 0
	}

	return duration
}

// hasDefaultCredentials checks if the database URL contains default credentials
func hasDefaultCredentials(databaseURL string) bool {
	// Check for the default postgres:postgres credentials
	return strings.Contains(databaseURL, "postgres:postgres@")
}

// bindsToNonLocalhost checks if the server address binds to a non-localhost interface
func bindsToNonLocalhost(addr string) bool {
	// Remove leading colon if present (e.g., ":8080" -> "8080")
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		// If parsing fails, check if it's just a port number
		if strings.HasPrefix(addr, ":") {
			// ":8080" format - binds to all interfaces
			return true
		}
		// Try to parse as host:port
		return false
	}

	// Empty host means bind to all interfaces (0.0.0.0)
	if host == "" {
		return true
	}

	// Check if it's localhost or 127.0.0.1
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return false
	}

	// Any other host (including 0.0.0.0 or specific IPs) means non-localhost
	return true
}

// isLikelyDocker checks if the database URL suggests Docker usage (host is service name, not localhost)
func isLikelyDocker(databaseURL string) bool {
	parsedURL, err := url.Parse(databaseURL)
	if err != nil {
		return false
	}
	host, _, err := net.SplitHostPort(parsedURL.Host)
	if err != nil {
		// If parsing fails, try without port
		host = parsedURL.Host
	}
	if host == "" || host == "localhost" || host == "127.0.0.1" {
		return false
	}
	// Check if it's a loopback IP
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return false
	}
	// Docker service names are typically not localhost/127.0.0.1 (e.g., "postgres")
	return true
}

// warnAboutDefaultCredentials logs warnings about default credentials usage
func warnAboutDefaultCredentials(databaseURL, serverAddr string) {
	if !hasDefaultCredentials(databaseURL) {
		return
	}

	inDocker := isLikelyDocker(databaseURL)

	// Always warn if default credentials are detected
	log.Println("")
	log.Println("⚠️  WARNING: Default database credentials detected (postgres:postgres)")
	log.Println("   These credentials are fine for local development, but should be changed")
	log.Println("   for any deployment accessible from a network.")
	if inDocker {
		log.Println("")
		log.Println("   If running with Docker Compose, ensure your docker-compose.yaml:")
		log.Println(
			"   - Binds PostgreSQL port to 127.0.0.1 (not 0.0.0.0) for localhost-only access",
		)
		log.Println("   - Or uses strong credentials if network access is required")
		log.Println("   - See docker-compose.yaml comments for security configuration details")
	}
	log.Println("")

	// Stronger warning if also binding to non-localhost
	if bindsToNonLocalhost(serverAddr) {
		log.Println("")
		log.Println("🚨 SECURITY WARNING: Default credentials + non-localhost binding detected!")
		log.Println("   Your server is configured to accept connections from the network")
		log.Println("   but is using default database credentials.")
		log.Println("")
		log.Println("   This is a security risk if your deployment is accessible beyond localhost.")
		log.Println(
			"   Please change the database credentials in your DATABASE_URL environment variable.",
		)
		if inDocker {
			log.Println("")
			log.Println("   If using Docker Compose:")
			log.Println("   1. Update POSTGRES_USER and POSTGRES_PASSWORD in docker-compose.yaml")
			log.Println("   2. Update DATABASE_URL in all services that connect to the database")
			log.Println(
				"   3. Verify PostgreSQL port is bound correctly (127.0.0.1:5432 for localhost-only)",
			)
		}
		log.Println("   Example: postgres://username:strongpassword@host:5432/octobud")
		log.Println("")
	}
}
