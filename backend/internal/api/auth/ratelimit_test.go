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
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestRateLimiter_Allow(t *testing.T) {
	logger := zap.NewNop()
	limiter := NewRateLimiter(5, 1*time.Minute, logger)

	// First 5 attempts should be allowed
	for i := 0; i < 5; i++ {
		if !limiter.Allow("testuser") {
			t.Errorf("Expected attempt %d to be allowed", i+1)
		}
	}

	// 6th attempt should be blocked
	if limiter.Allow("testuser") {
		t.Error("Expected 6th attempt to be blocked")
	}

	// Different user should still be allowed
	if !limiter.Allow("otheruser") {
		t.Error("Expected different user to be allowed")
	}
}

func TestRateLimiter_Reset(t *testing.T) {
	logger := zap.NewNop()
	limiter := NewRateLimiter(5, 1*time.Minute, logger)

	// Exhaust attempts
	for i := 0; i < 5; i++ {
		limiter.Allow("testuser")
	}

	// Should be blocked
	if limiter.Allow("testuser") {
		t.Error("Expected attempt to be blocked")
	}

	// Reset should allow again
	limiter.Reset("testuser")
	if !limiter.Allow("testuser") {
		t.Error("Expected attempt to be allowed after reset")
	}
}
