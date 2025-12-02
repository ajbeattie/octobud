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
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

func TestCSRFMiddleware_ValidToken(t *testing.T) {
	logger := zap.NewNop()
	middleware := CSRFMiddleware(logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware(handler)

	req := httptest.NewRequest("POST", "/test", http.NoBody)

	token := "test-csrf-token"
	req.AddCookie(&http.Cookie{
		Name:  csrfCookieName,
		Value: token,
	})
	req.Header.Set(csrfHeaderName, token)

	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestCSRFMiddleware_MissingCookie(t *testing.T) {
	logger := zap.NewNop()
	middleware := CSRFMiddleware(logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware(handler)

	req := httptest.NewRequest("POST", "/test", http.NoBody)
	req.Header.Set(csrfHeaderName, "test-token")

	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}
}

func TestCSRFMiddleware_MissingHeader(t *testing.T) {
	logger := zap.NewNop()
	middleware := CSRFMiddleware(logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware(handler)

	req := httptest.NewRequest("POST", "/test", http.NoBody)
	req.AddCookie(&http.Cookie{
		Name:  csrfCookieName,
		Value: "test-token",
	})

	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}
}

func TestCSRFMiddleware_TokenMismatch(t *testing.T) {
	logger := zap.NewNop()
	middleware := CSRFMiddleware(logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware(handler)

	req := httptest.NewRequest("POST", "/test", http.NoBody)
	req.AddCookie(&http.Cookie{
		Name:  csrfCookieName,
		Value: "cookie-token",
	})
	req.Header.Set(csrfHeaderName, "header-token")

	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}
}

func TestCSRFMiddleware_SafeMethods(t *testing.T) {
	logger := zap.NewNop()
	middleware := CSRFMiddleware(logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware(handler)

	methods := []string{"GET", "HEAD", "OPTIONS"}
	for _, method := range methods {
		req := httptest.NewRequest(method, "/test", http.NoBody)
		w := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for %s, got %d", method, w.Code)
		}
	}
}

func TestConstantTimeCompare(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected bool
	}{
		{"equal strings", "test", "test", true},
		{"different strings", "test", "other", false},
		{"different lengths", "test", "testing", false},
		{"empty strings", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := constantTimeCompare(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf(
					"constantTimeCompare(%q, %q) = %v, expected %v",
					tt.a,
					tt.b,
					result,
					tt.expected,
				)
			}
		})
	}
}
