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

package auth

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

// RateLimiter tracks login attempts per username to prevent brute force attacks
type RateLimiter struct {
	attempts map[string][]time.Time
	mu       sync.RWMutex
	limit    int
	window   time.Duration
	logger   *zap.Logger
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(limit int, window time.Duration, logger *zap.Logger) *RateLimiter {
	rl := &RateLimiter{
		attempts: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
		logger:   logger,
	}

	// Start background cleanup goroutine
	go rl.cleanup()

	return rl
}

// Allow checks if a request from the given username should be allowed
// Returns true if allowed, false if rate limit exceeded
func (rl *RateLimiter) Allow(username string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Get or create attempts list for this username
	attempts, exists := rl.attempts[username]
	if !exists {
		attempts = make([]time.Time, 0, rl.limit)
	}

	// Remove attempts outside the time window
	validAttempts := make([]time.Time, 0, len(attempts))
	for _, attempt := range attempts {
		if attempt.After(cutoff) {
			validAttempts = append(validAttempts, attempt)
		}
	}

	// Check if limit exceeded
	if len(validAttempts) >= rl.limit {
		rl.attempts[username] = validAttempts
		return false
	}

	// Add current attempt
	validAttempts = append(validAttempts, now)
	rl.attempts[username] = validAttempts

	return true
}

// Reset removes all attempts for a username (called on successful login)
func (rl *RateLimiter) Reset(username string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	delete(rl.attempts, username)
}

// cleanup periodically removes old entries to prevent memory leaks
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		cutoff := now.Add(-rl.window)

		for username, attempts := range rl.attempts {
			validAttempts := make([]time.Time, 0, len(attempts))
			for _, attempt := range attempts {
				if attempt.After(cutoff) {
					validAttempts = append(validAttempts, attempt)
				}
			}

			if len(validAttempts) == 0 {
				delete(rl.attempts, username)
			} else {
				rl.attempts[username] = validAttempts
			}
		}

		rl.mu.Unlock()
	}
}

// RateLimitMiddleware creates middleware that rate limits login attempts per username
func RateLimitMiddleware(limiter *RateLimiter, logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Read request body to extract username
			var loginReq struct {
				Username string `json:"username"`
			}

			// Read and restore body
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				logger.Debug("rate limit: failed to read request body", zap.Error(err))
				http.Error(w, "Invalid request", http.StatusBadRequest)
				return
			}
			_ = r.Body.Close() // Body already read, safe to ignore error

			// Parse JSON to get username
			if err := json.Unmarshal(bodyBytes, &loginReq); err != nil {
				logger.Debug("rate limit: failed to parse request body", zap.Error(err))
				http.Error(w, "Invalid request", http.StatusBadRequest)
				return
			}

			// Check rate limit
			if !limiter.Allow(loginReq.Username) {
				logger.Debug("rate limit exceeded", zap.String("username", loginReq.Username))
				http.Error(
					w,
					"Too many login attempts. Please try again later.",
					http.StatusTooManyRequests,
				)
				return
			}

			// Restore body for the handler
			r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

			next.ServeHTTP(w, r)
		})
	}
}
