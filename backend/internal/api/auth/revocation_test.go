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

func TestTokenRevocation_RevokeToken(t *testing.T) {
	logger := zap.NewNop()
	tr := NewTokenRevocation(logger)

	token := "test-token"
	expiresAt := time.Now().Add(1 * time.Hour)

	// Token should not be revoked initially
	if tr.IsTokenRevoked(token) {
		t.Error("Expected token to not be revoked initially")
	}

	// Revoke token
	tr.RevokeToken(token, expiresAt)

	// Token should now be revoked
	if !tr.IsTokenRevoked(token) {
		t.Error("Expected token to be revoked")
	}
}

func TestTokenRevocation_ExpiredToken(t *testing.T) {
	logger := zap.NewNop()
	tr := NewTokenRevocation(logger)

	token := "test-token"
	expiresAt := time.Now().Add(-1 * time.Hour) // Already expired

	// Revoke with expired time
	tr.RevokeToken(token, expiresAt)

	// Token should not be considered revoked (cleanup will remove it)
	// But IsTokenRevoked checks expiration, so it should return false
	if tr.IsTokenRevoked(token) {
		t.Error("Expected expired token to not be considered revoked")
	}
}
