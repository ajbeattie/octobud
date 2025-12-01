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
	"context"

	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	authsvc "github.com/ajbeattie/octobud/backend/internal/core/auth"
	"github.com/ajbeattie/octobud/backend/internal/core/auth/mocks"
	"github.com/ajbeattie/octobud/backend/internal/models"
)

func setupTestHandler(ctrl *gomock.Controller) (*Handler, *mocks.MockAuthService) {
	logger := zap.NewNop()
	mockService := mocks.NewMockAuthService(ctrl)
	jwtSecret := "test-secret-key-for-testing-only"
	tokenExpiry := 24 * time.Hour // Use 24h for tests
	rateLimiter := NewRateLimiter(5, 1*time.Minute, logger)
	tokenRevocation := NewTokenRevocation(logger)
	return New(
		logger,
		mockService,
		jwtSecret,
		tokenExpiry,
		rateLimiter,
		tokenRevocation,
		false,
	), mockService
}

func createRequest(method, url string, body interface{}) *http.Request {
	var reqBody []byte
	if body != nil {
		var err error
		reqBody, err = json.Marshal(body)
		if err != nil {
			reqBody = nil
		}
	}
	req := httptest.NewRequest(method, url, bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestHandler_HandleLogin(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*mocks.MockAuthService)
		expectedStatus int
		expectedBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "success returns token and username",
			requestBody: LoginRequest{
				Username: "admin",
				Password: "admin",
			},
			setupMock: func(m *mocks.MockAuthService) {
				hash, hashErr := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
				require.NoError(t, hashErr)
				user := &models.User{
					ID:           1,
					Username:     "admin",
					PasswordHash: string(hash),
				}
				m.EXPECT().ValidatePassword(gomock.Any(), "admin", "admin").Return(nil)
				m.EXPECT().GetUser(gomock.Any()).Return(user, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response LoginResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.NotEmpty(t, response.Token)
				require.Equal(t, "admin", response.Username)

				// Verify token is valid
				token, err := jwt.Parse(
					response.Token,
					func(_ *jwt.Token) (interface{}, error) {
						return []byte("test-secret-key-for-testing-only"), nil
					},
				)
				require.NoError(t, err)
				assert.True(t, token.Valid)
			},
		},
		{
			name: "invalid credentials returns 401",
			requestBody: LoginRequest{
				Username: "admin",
				Password: "wrongpassword",
			},
			setupMock: func(m *mocks.MockAuthService) {
				m.EXPECT().
					ValidatePassword(gomock.Any(), "admin", "wrongpassword").
					Return(authsvc.ErrInvalidPassword)
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Contains(t, response.Error, "Invalid username or password")
			},
		},
		{
			name: "missing username returns 400",
			requestBody: LoginRequest{
				Username: "",
				Password: "admin",
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Contains(t, response.Error, "required")
			},
		},
		{
			name: "missing password returns 400",
			requestBody: LoginRequest{
				Username: "admin",
				Password: "",
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Contains(t, response.Error, "required")
			},
		},
		{
			name:           "invalid body returns 400",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "service error returns 500",
			requestBody: LoginRequest{
				Username: "admin",
				Password: "admin",
			},
			setupMock: func(m *mocks.MockAuthService) {
				m.EXPECT().
					ValidatePassword(gomock.Any(), "admin", "admin").
					Return(errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			handler, mockService := setupTestHandler(ctrl)
			if tt.setupMock != nil {
				tt.setupMock(mockService)
			}

			req := createRequest(http.MethodPost, "/api/auth/login", tt.requestBody)
			w := httptest.NewRecorder()

			handler.HandleLogin(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != nil {
				tt.expectedBody(t, w)
			}
		})
	}
}

func TestHandler_HandleGetCurrentUser(t *testing.T) {
	tests := []struct {
		name           string
		username       string
		setupContext   func(*http.Request) *http.Request
		expectedStatus int
		expectedBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:     "success returns username",
			username: "admin",
			setupContext: func(req *http.Request) *http.Request {
				ctx := context.WithValue(req.Context(), usernameContextKey, "admin")
				return req.WithContext(ctx)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response UserResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "admin", response.Username)
			},
		},
		{
			name:     "missing username in context returns 401",
			username: "",
			setupContext: func(req *http.Request) *http.Request {
				return req
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			handler, _ := setupTestHandler(ctrl)

			req := createRequest(http.MethodGet, "/api/auth/me", nil)
			req = tt.setupContext(req)
			w := httptest.NewRecorder()

			handler.HandleGetCurrentUser(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != nil {
				tt.expectedBody(t, w)
			}
		})
	}
}

func TestHandler_HandleUpdateCredentials(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*mocks.MockAuthService)
		setupContext   func(*http.Request) *http.Request
		expectedStatus int
		expectedBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "success updates username only",
			requestBody: UpdateCredentialsRequest{
				CurrentPassword: "admin",
				NewUsername:     stringPtr("newadmin"),
			},
			setupMock: func(m *mocks.MockAuthService) {
				// ValidatePassword is mocked, so it won't actually call GetUser
				// We just need to mock ValidatePassword to return success
				m.EXPECT().ValidatePassword(gomock.Any(), "admin", "admin").Return(nil)
				m.EXPECT().UpdateUsername(gomock.Any(), "newadmin").Return(nil)
			},
			setupContext: func(req *http.Request) *http.Request {
				ctx := context.WithValue(req.Context(), usernameContextKey, "admin")
				return req.WithContext(ctx)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response UserResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "newadmin", response.Username)
			},
		},
		{
			name: "success updates password only",
			requestBody: UpdateCredentialsRequest{
				CurrentPassword: "admin",
				NewPassword:     stringPtr("newpassword"),
			},
			setupMock: func(m *mocks.MockAuthService) {
				// ValidatePassword is mocked, handler also calls GetUser to get current username
				hash, hashErr := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
				require.NoError(t, hashErr)
				user := &models.User{
					ID:           1,
					Username:     "admin",
					PasswordHash: string(hash),
				}
				// ValidatePassword is mocked (won't call GetUser), but handler calls GetUser to get current username
				m.EXPECT().ValidatePassword(gomock.Any(), "admin", "admin").Return(nil)
				m.EXPECT().
					GetUser(gomock.Any()).
					Return(user, nil)
					// Called by handler to get current username
				m.EXPECT().UpdatePassword(gomock.Any(), "newpassword").Return(nil)
			},
			setupContext: func(req *http.Request) *http.Request {
				ctx := context.WithValue(req.Context(), usernameContextKey, "admin")
				return req.WithContext(ctx)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response UserResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "admin", response.Username)
			},
		},
		{
			name: "success updates both username and password",
			requestBody: UpdateCredentialsRequest{
				CurrentPassword: "admin",
				NewUsername:     stringPtr("newadmin"),
				NewPassword:     stringPtr("newpassword"),
			},
			setupMock: func(m *mocks.MockAuthService) {
				m.EXPECT().ValidatePassword(gomock.Any(), "admin", "admin").Return(nil)
				m.EXPECT().UpdateCredentials(gomock.Any(), "newadmin", "newpassword").Return(nil)
			},
			setupContext: func(req *http.Request) *http.Request {
				ctx := context.WithValue(req.Context(), usernameContextKey, "admin")
				return req.WithContext(ctx)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response UserResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "newadmin", response.Username)
			},
		},
		{
			name: "missing current password returns 400",
			requestBody: UpdateCredentialsRequest{
				CurrentPassword: "",
				NewUsername:     stringPtr("newadmin"),
			},
			setupContext: func(req *http.Request) *http.Request {
				ctx := context.WithValue(req.Context(), usernameContextKey, "admin")
				return req.WithContext(ctx)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing both new username and password returns 400",
			requestBody: UpdateCredentialsRequest{
				CurrentPassword: "admin",
			},
			setupContext: func(req *http.Request) *http.Request {
				ctx := context.WithValue(req.Context(), usernameContextKey, "admin")
				return req.WithContext(ctx)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid current password returns 401",
			requestBody: UpdateCredentialsRequest{
				CurrentPassword: "wrongpassword",
				NewUsername:     stringPtr("newadmin"),
			},
			setupMock: func(m *mocks.MockAuthService) {
				m.EXPECT().
					ValidatePassword(gomock.Any(), "admin", "wrongpassword").
					Return(authsvc.ErrInvalidPassword)
			},
			setupContext: func(req *http.Request) *http.Request {
				ctx := context.WithValue(req.Context(), usernameContextKey, "admin")
				return req.WithContext(ctx)
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "missing username in context returns 401",
			requestBody: UpdateCredentialsRequest{
				CurrentPassword: "admin",
				NewUsername:     stringPtr("newadmin"),
			},
			setupContext: func(req *http.Request) *http.Request {
				return req
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "password too short returns 400",
			requestBody: UpdateCredentialsRequest{
				CurrentPassword: "admin",
				NewPassword:     stringPtr("short"),
			},
			setupMock: func(m *mocks.MockAuthService) {
				hash, hashErr := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
				require.NoError(t, hashErr)
				user := &models.User{
					ID:           1,
					Username:     "admin",
					PasswordHash: string(hash),
				}
				m.EXPECT().ValidatePassword(gomock.Any(), "admin", "admin").Return(nil)
				m.EXPECT().GetUser(gomock.Any()).Return(user, nil)
			},
			setupContext: func(req *http.Request) *http.Request {
				ctx := context.WithValue(req.Context(), usernameContextKey, "admin")
				return req.WithContext(ctx)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "password too long returns 400",
			requestBody: UpdateCredentialsRequest{
				CurrentPassword: "admin",
				NewPassword:     stringPtr(strings.Repeat("a", 129)),
			},
			setupMock: func(m *mocks.MockAuthService) {
				hash, hashErr := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
				require.NoError(t, hashErr)
				user := &models.User{
					ID:           1,
					Username:     "admin",
					PasswordHash: string(hash),
				}
				m.EXPECT().ValidatePassword(gomock.Any(), "admin", "admin").Return(nil)
				m.EXPECT().GetUser(gomock.Any()).Return(user, nil)
			},
			setupContext: func(req *http.Request) *http.Request {
				ctx := context.WithValue(req.Context(), usernameContextKey, "admin")
				return req.WithContext(ctx)
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			handler, mockService := setupTestHandler(ctrl)
			if tt.setupMock != nil {
				tt.setupMock(mockService)
			}

			req := createRequest(http.MethodPut, "/api/auth/credentials", tt.requestBody)
			req = tt.setupContext(req)
			w := httptest.NewRecorder()

			handler.HandleUpdateCredentials(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != nil {
				tt.expectedBody(t, w)
			}
		})
	}
}

func TestJWTMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		setupRequest   func() *http.Request
		expectedStatus int
	}{
		{
			name: "valid token allows request",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/api/test", http.NoBody)
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
					"username": "admin",
					"exp":      time.Now().Add(24 * time.Hour).Unix(),
					"iat":      time.Now().Unix(),
				})
				tokenString, signErr := token.SignedString(
					[]byte("test-secret-key-for-testing-only"),
				)
				if signErr != nil {
					panic(signErr) // Should never happen in tests
				}
				req.Header.Set("Authorization", "Bearer "+tokenString)
				return req
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "missing authorization header returns 401",
			setupRequest: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/api/test", http.NoBody)
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "invalid token format returns 401",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/api/test", http.NoBody)
				req.Header.Set("Authorization", "InvalidFormat token")
				return req
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "invalid token signature returns 401",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/api/test", http.NoBody)
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
					"username": "admin",
					"exp":      time.Now().Add(24 * time.Hour).Unix(),
					"iat":      time.Now().Unix(),
				})
				tokenString, signErr := token.SignedString([]byte("wrong-secret"))
				if signErr != nil {
					panic(signErr) // Should never happen in tests
				}
				req.Header.Set("Authorization", "Bearer "+tokenString)
				return req
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zap.NewNop()
			jwtSecret := "test-secret-key-for-testing-only"

			handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			tokenRevocation := NewTokenRevocation(logger)
			middleware := JWTMiddleware(jwtSecret, logger, tokenRevocation)
			wrappedHandler := middleware(handler)

			req := tt.setupRequest()
			w := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

type errorResponse struct {
	Error string `json:"error"`
}
