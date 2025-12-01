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
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"

	"github.com/ajbeattie/octobud/backend/internal/api/shared"
	authsvc "github.com/ajbeattie/octobud/backend/internal/core/auth"
)

// Handler handles authentication-related HTTP routes
type Handler struct {
	logger          *zap.Logger
	authSvc         authsvc.AuthService
	jwtSecret       string
	tokenExpiry     time.Duration
	rateLimiter     *RateLimiter
	tokenRevocation *TokenRevocation
	secureCookies   bool // Force secure cookies (overrides auto-detection if true)
}

// New creates a new auth handler
func New(
	logger *zap.Logger,
	authSvc authsvc.AuthService,
	jwtSecret string,
	tokenExpiry time.Duration,
	rateLimiter *RateLimiter,
	tokenRevocation *TokenRevocation,
	secureCookies bool,
) *Handler {
	return &Handler{
		logger:          logger,
		authSvc:         authSvc,
		jwtSecret:       jwtSecret,
		tokenExpiry:     tokenExpiry,
		rateLimiter:     rateLimiter,
		tokenRevocation: tokenRevocation,
		secureCookies:   secureCookies,
	}
}

// Register registers auth routes on the provided router
func (h *Handler) Register(r chi.Router) {
	r.Route("/auth", func(r chi.Router) {
		r.Post("/login", h.HandleLogin)
		r.Get("/me", h.HandleGetCurrentUser)
		r.Post("/refresh", h.HandleRefreshToken)
		r.Post("/logout", h.HandleLogout)
		r.Put("/credentials", h.HandleUpdateCredentials)
	})
}

// HandleLogin handles POST /api/auth/login
func (h *Handler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Debug("failed to decode login request", zap.Error(err))
		shared.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Username == "" || req.Password == "" {
		shared.WriteError(w, http.StatusBadRequest, "Username and password are required")
		return
	}

	// Validate credentials
	ctx := r.Context()
	if err := h.authSvc.ValidatePassword(ctx, req.Username, req.Password); err != nil {
		if errors.Is(err, authsvc.ErrInvalidPassword) || errors.Is(err, authsvc.ErrUserNotFound) {
			// Security audit log: failed login attempt
			h.logger.Info("failed login attempt",
				zap.String("username", req.Username),
				zap.String("ip", getClientIP(r)),
			)
			shared.WriteError(w, http.StatusUnauthorized, "Invalid username or password")
			return
		}
		h.logger.Error("failed to validate password", zap.Error(err))
		shared.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Reset rate limiter on successful login
	if h.rateLimiter != nil {
		h.rateLimiter.Reset(req.Username)
	}

	// Get user to get username
	user, err := h.authSvc.GetUser(ctx)
	if err != nil {
		h.logger.Error("failed to get user", zap.Error(err))
		shared.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Security audit log: successful login
	h.logger.Info("successful login",
		zap.String("username", user.Username),
		zap.String("ip", getClientIP(r)),
	)

	// Generate JWT token
	expiresAt := time.Now().Add(h.tokenExpiry)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": user.Username,
		"exp":      expiresAt.Unix(),
		"iat":      time.Now().Unix(),
	})

	tokenString, err := token.SignedString([]byte(h.jwtSecret))
	if err != nil {
		h.logger.Error("failed to sign token", zap.Error(err))
		shared.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Generate and set CSRF token cookie
	csrfToken, err := GenerateCSRFToken()
	if err != nil {
		h.logger.Error("failed to generate CSRF token", zap.Error(err))
		shared.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	SetCSRFCookie(w, csrfToken, h.secureCookies, r)

	response := LoginResponse{
		Token:     tokenString,
		Username:  user.Username,
		CSRFToken: csrfToken, // Include in response so frontend can send it in header
	}

	shared.WriteJSON(w, http.StatusOK, response)
}

// HandleRefreshToken handles POST /api/auth/refresh
// Validates the current token and issues a new one with extended expiration
func (h *Handler) HandleRefreshToken(w http.ResponseWriter, r *http.Request) {
	username := GetUsernameFromContext(r.Context())
	if username == "" {
		shared.WriteError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get user to verify they still exist
	ctx := r.Context()
	user, err := h.authSvc.GetUser(ctx)
	if err != nil {
		h.logger.Error("failed to get user for refresh", zap.Error(err))
		shared.WriteError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Generate new JWT token with extended expiration
	expiresAt := time.Now().Add(h.tokenExpiry)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": user.Username,
		"exp":      expiresAt.Unix(),
		"iat":      time.Now().Unix(),
	})

	tokenString, err := token.SignedString([]byte(h.jwtSecret))
	if err != nil {
		h.logger.Error("failed to sign refresh token", zap.Error(err))
		shared.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Generate new CSRF token on refresh
	csrfToken, err := GenerateCSRFToken()
	if err != nil {
		h.logger.Error("failed to generate CSRF token", zap.Error(err))
		shared.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	SetCSRFCookie(w, csrfToken, h.secureCookies, r)

	response := LoginResponse{
		Token:     tokenString,
		Username:  user.Username,
		CSRFToken: csrfToken,
	}

	shared.WriteJSON(w, http.StatusOK, response)
}

// HandleLogout handles POST /api/auth/logout
// Revokes the current JWT token by adding it to the blacklist
func (h *Handler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	// Extract token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		shared.WriteError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Extract token from "Bearer <token>"
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		shared.WriteError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	tokenString := parts[1]

	// Parse token to get expiration time
	token, err := jwt.Parse(tokenString, func(_ *jwt.Token) (interface{}, error) {
		return []byte(h.jwtSecret), nil
	})

	if err != nil || !token.Valid {
		// Token is already invalid, but we'll still try to revoke it
		// Use a default expiration (current time + token expiry) as fallback
		expiresAt := time.Now().Add(h.tokenExpiry)
		if h.tokenRevocation != nil {
			h.tokenRevocation.RevokeToken(tokenString, expiresAt)
		}
		shared.WriteJSON(w, http.StatusOK, map[string]string{"message": "Logged out"})
		return
	}

	// Extract expiration from claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		expiresAt := time.Now().Add(h.tokenExpiry)
		if h.tokenRevocation != nil {
			h.tokenRevocation.RevokeToken(tokenString, expiresAt)
		}
		shared.WriteJSON(w, http.StatusOK, map[string]string{"message": "Logged out"})
		return
	}

	// Get expiration time from token
	var expiresAt time.Time
	if exp, ok := claims["exp"].(float64); ok {
		expiresAt = time.Unix(int64(exp), 0)
	} else {
		// Fallback to current time + token expiry
		expiresAt = time.Now().Add(h.tokenExpiry)
	}

	// Revoke token
	if h.tokenRevocation != nil {
		h.tokenRevocation.RevokeToken(tokenString, expiresAt)
	}

	// Security audit log: logout
	username := GetUsernameFromContext(r.Context())
	h.logger.Info("user logged out",
		zap.String("username", username),
		zap.String("ip", getClientIP(r)),
	)

	// Clear CSRF cookie (must match Secure flag of original cookie)
	secure := h.secureCookies
	if !secure {
		secure = shared.IsHTTPS(r)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   secure,
		MaxAge:   -1, // Delete cookie
	})

	shared.WriteJSON(w, http.StatusOK, map[string]string{"message": "Logged out"})
}

// HandleGetCurrentUser handles GET /api/auth/me
func (h *Handler) HandleGetCurrentUser(w http.ResponseWriter, r *http.Request) {
	username := GetUsernameFromContext(r.Context())
	if username == "" {
		shared.WriteError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	response := UserResponse{
		Username: username,
	}

	shared.WriteJSON(w, http.StatusOK, response)
}

// HandleUpdateCredentials handles PUT /api/auth/credentials
func (h *Handler) HandleUpdateCredentials(w http.ResponseWriter, r *http.Request) {
	username := GetUsernameFromContext(r.Context())
	if username == "" {
		shared.WriteError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req UpdateCredentialsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Debug("failed to decode update credentials request", zap.Error(err))
		shared.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.CurrentPassword == "" {
		shared.WriteError(w, http.StatusBadRequest, "Current password is required")
		return
	}

	// At least one of new username or new password must be provided
	if req.NewUsername == nil && req.NewPassword == nil {
		shared.WriteError(
			w,
			http.StatusBadRequest,
			"At least one of newUsername or newPassword must be provided",
		)
		return
	}

	ctx := r.Context()

	// Validate current password
	if err := h.authSvc.ValidatePassword(ctx, username, req.CurrentPassword); err != nil {
		if errors.Is(err, authsvc.ErrInvalidPassword) {
			h.logger.Debug("invalid current password")
			shared.WriteError(w, http.StatusUnauthorized, "Invalid current password")
			return
		}
		h.logger.Error("failed to validate current password", zap.Error(err))
		shared.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Update credentials
	var newUsername string
	if req.NewUsername != nil {
		// Validate username format
		if err := authsvc.ValidateUsername(*req.NewUsername); err != nil {
			h.logger.Debug("username validation failed", zap.Error(err))
			shared.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		newUsername = *req.NewUsername
	} else {
		// Keep current username
		user, err := h.authSvc.GetUser(ctx)
		if err != nil {
			h.logger.Error("failed to get user", zap.Error(err))
			shared.WriteError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
		newUsername = user.Username
	}

	if req.NewPassword != nil {
		// Validate password strength
		if err := authsvc.ValidatePasswordStrength(*req.NewPassword); err != nil {
			h.logger.Debug("password validation failed", zap.Error(err))
			shared.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		// Update both username and password
		if req.NewUsername != nil {
			if err := h.authSvc.UpdateCredentials(ctx, newUsername, *req.NewPassword); err != nil {
				h.logger.Error("failed to update credentials", zap.Error(err))
				shared.WriteError(w, http.StatusInternalServerError, "Internal server error")
				return
			}
		} else {
			// Update password only
			if err := h.authSvc.UpdatePassword(ctx, *req.NewPassword); err != nil {
				h.logger.Error("failed to update password", zap.Error(err))
				shared.WriteError(w, http.StatusInternalServerError, "Internal server error")
				return
			}
		}
	} else {
		// Update username only
		if err := h.authSvc.UpdateUsername(ctx, newUsername); err != nil {
			h.logger.Error("failed to update username", zap.Error(err))
			shared.WriteError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
	}

	// Security audit log: credentials updated
	oldUsername := username
	h.logger.Info("credentials updated",
		zap.String("old_username", oldUsername),
		zap.String("new_username", newUsername),
		zap.Bool("password_changed", req.NewPassword != nil),
		zap.String("ip", getClientIP(r)),
	)

	response := UserResponse{
		Username: newUsername,
	}

	shared.WriteJSON(w, http.StatusOK, response)
}

// getClientIP extracts the client IP address from the request
// Checks X-Forwarded-For, X-Real-IP, and RemoteAddr
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (set by reverse proxies)
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header (set by some reverse proxies)
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return strings.TrimSpace(realIP)
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
