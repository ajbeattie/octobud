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
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

type contextKey string

const usernameContextKey contextKey = "username"

// GetUsernameFromContext retrieves the username from the request context
func GetUsernameFromContext(ctx context.Context) string {
	if username, ok := ctx.Value(usernameContextKey).(string); ok {
		return username
	}
	return ""
}

// SetUsernameInContext adds the username to a context.
// This is primarily used for testing purposes.
func SetUsernameInContext(ctx context.Context, username string) context.Context {
	return context.WithValue(ctx, usernameContextKey, username)
}

// JWTMiddleware validates JWT tokens and attaches username to request context
func JWTMiddleware(
	jwtSecret string,
	logger *zap.Logger,
	tokenRevocation *TokenRevocation,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				logger.Debug("missing authorization header")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Extract token from "Bearer <token>"
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				logger.Debug("invalid authorization header format")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			tokenString := parts[1]

			// Check if token is revoked
			if tokenRevocation != nil && tokenRevocation.IsTokenRevoked(tokenString) {
				logger.Debug("token has been revoked")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Parse and validate token
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				// Validate signing method
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(jwtSecret), nil
			})

			if err != nil {
				logger.Debug("failed to parse token", zap.Error(err))
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if !token.Valid {
				logger.Debug("invalid token")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Extract username from claims
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				logger.Debug("invalid token claims")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			username, ok := claims["username"].(string)
			if !ok || username == "" {
				logger.Debug("missing username in token claims")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Attach username to context
			ctx := context.WithValue(r.Context(), usernameContextKey, username)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
