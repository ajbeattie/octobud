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

// Package auth provides the authentication service.
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"go.uber.org/zap"

	"github.com/ajbeattie/octobud/backend/internal/api/shared"
)

const (
	csrfCookieName   = "csrf_token"
	csrfHeaderName   = "X-CSRF-Token"
	csrfCookieMaxAge = 7 * 24 * 3600 // 7 days in seconds
)

// GenerateCSRFToken generates a cryptographically secure random CSRF token
func GenerateCSRFToken() (string, error) {
	bytes := make([]byte, 32) // 256 bits
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// SetCSRFCookie sets the CSRF token in an httpOnly cookie
// If r is provided, it will auto-detect HTTPS and set Secure accordingly.
// If forceSecure is true, Secure will always be true regardless of request.
// If forceSecure is false and r is nil, Secure will be false.
func SetCSRFCookie(w http.ResponseWriter, token string, forceSecure bool, r *http.Request) {
	// Determine if cookie should be secure
	secure := forceSecure
	if !secure && r != nil {
		// Auto-detect HTTPS from request
		secure = shared.IsHTTPS(r)
	}

	// Use Lax SameSite for development (allows cross-origin requests on same domain)
	// In production with HTTPS, this still provides good CSRF protection
	// Strict would prevent cookies from being sent on cross-origin requests
	sameSite := http.SameSiteLaxMode
	if secure {
		// In production with HTTPS, we can use Strict
		sameSite = http.SameSiteStrictMode
	}

	cookie := &http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: sameSite,
		MaxAge:   csrfCookieMaxAge,
		Secure:   secure, // Auto-detected from request or forced via flag
	}
	http.SetCookie(w, cookie)
}

// GetCSRFCookie retrieves the CSRF token from the cookie
func GetCSRFCookie(r *http.Request) string {
	cookie, err := r.Cookie(csrfCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// CSRFMiddleware validates CSRF tokens using the double-submit cookie pattern
// Requires X-CSRF-Token header to match the csrf_token cookie value
func CSRFMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip CSRF check for safe methods (GET, HEAD, OPTIONS)
			if r.Method == http.MethodGet || r.Method == http.MethodHead ||
				r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			// Get CSRF token from cookie
			cookieToken := GetCSRFCookie(r)
			if cookieToken == "" {
				logger.Debug("CSRF: missing cookie token")
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			// Get CSRF token from header
			headerToken := r.Header.Get(csrfHeaderName)
			if headerToken == "" {
				logger.Debug("CSRF: missing header token")
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			// Compare tokens using constant-time comparison
			if !constantTimeCompare(cookieToken, headerToken) {
				logger.Debug("CSRF: token mismatch")
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// constantTimeCompare performs a constant-time comparison of two strings
// This prevents timing attacks when comparing CSRF tokens
func constantTimeCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}

	result := 0
	for i := 0; i < len(a); i++ {
		result |= int(a[i]) ^ int(b[i])
	}

	return result == 0
}
