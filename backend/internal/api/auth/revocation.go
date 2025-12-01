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
	"sync"
	"time"

	"go.uber.org/zap"
)

// TokenRevocation manages a blacklist of revoked JWT tokens
type TokenRevocation struct {
	blacklist map[string]time.Time
	mu        sync.RWMutex
	logger    *zap.Logger
}

// NewTokenRevocation creates a new token revocation manager
func NewTokenRevocation(logger *zap.Logger) *TokenRevocation {
	tr := &TokenRevocation{
		blacklist: make(map[string]time.Time),
		logger:    logger,
	}

	// Start background cleanup goroutine
	go tr.cleanup()

	return tr
}

// RevokeToken adds a token to the blacklist until its expiration time
func (tr *TokenRevocation) RevokeToken(token string, expiresAt time.Time) {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	tr.blacklist[token] = expiresAt
	tr.logger.Debug("token revoked", zap.Time("expires_at", expiresAt))
}

// IsTokenRevoked checks if a token is in the blacklist
func (tr *TokenRevocation) IsTokenRevoked(token string) bool {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	expiresAt, exists := tr.blacklist[token]
	if !exists {
		return false
	}

	// If token has expired, it's effectively not revoked (cleanup will remove it)
	// But we still check to be safe
	if time.Now().After(expiresAt) {
		return false
	}

	return true
}

// cleanup periodically removes expired tokens from the blacklist
func (tr *TokenRevocation) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		tr.mu.Lock()
		now := time.Now()

		for token, expiresAt := range tr.blacklist {
			if now.After(expiresAt) {
				delete(tr.blacklist, token)
			}
		}

		tr.mu.Unlock()
	}
}
