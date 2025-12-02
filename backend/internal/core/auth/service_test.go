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
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"

	"github.com/ajbeattie/octobud/backend/internal/db"
	"github.com/ajbeattie/octobud/backend/internal/db/mocks"
)

func setupTestService(ctrl *gomock.Controller) (*Service, *mocks.MockStore) {
	mockStore := mocks.NewMockStore(ctrl)
	service := NewService(mockStore)
	return service, mockStore
}

func TestService_GetUser(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(*mocks.MockStore)
		expectError bool
		expectUser  bool
	}{
		{
			name: "success returns user",
			setupMock: func(m *mocks.MockStore) {
				hash, hashErr := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
				require.NoError(t, hashErr)
				m.EXPECT().GetUser(gomock.Any()).Return(db.User{
					ID:           1,
					Username:     "admin",
					PasswordHash: string(hash),
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				}, nil)
			},
			expectError: false,
			expectUser:  true,
		},
		{
			name: "user not found returns error",
			setupMock: func(m *mocks.MockStore) {
				m.EXPECT().GetUser(gomock.Any()).Return(db.User{}, sql.ErrNoRows)
			},
			expectError: true,
			expectUser:  false,
		},
		{
			name: "database error returns error",
			setupMock: func(m *mocks.MockStore) {
				m.EXPECT().GetUser(gomock.Any()).Return(db.User{}, errors.New("database error"))
			},
			expectError: true,
			expectUser:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			service, mockStore := setupTestService(ctrl)
			tt.setupMock(mockStore)

			user, err := service.GetUser(context.Background())

			if tt.expectError {
				require.Error(t, err)
				require.Nil(t, user)
			} else {
				require.NoError(t, err)
				require.NotNil(t, user)
				if tt.expectUser {
					assert.Equal(t, "admin", user.Username)
				}
			}
		})
	}
}

func TestService_ValidatePassword(t *testing.T) {
	tests := []struct {
		name        string
		username    string
		password    string
		setupMock   func(*mocks.MockStore)
		expectError bool
	}{
		{
			name:     "valid credentials",
			username: "admin",
			password: "password",
			setupMock: func(m *mocks.MockStore) {
				hash, hashErr := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
				require.NoError(t, hashErr)
				m.EXPECT().GetUser(gomock.Any()).Return(db.User{
					ID:           1,
					Username:     "admin",
					PasswordHash: string(hash),
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				}, nil)
			},
			expectError: false,
		},
		{
			name:     "invalid password",
			username: "admin",
			password: "wrongpassword",
			setupMock: func(m *mocks.MockStore) {
				hash, hashErr := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
				require.NoError(t, hashErr)
				m.EXPECT().GetUser(gomock.Any()).Return(db.User{
					ID:           1,
					Username:     "admin",
					PasswordHash: string(hash),
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				}, nil)
			},
			expectError: true,
		},
		{
			name:     "invalid username",
			username: "wronguser",
			password: "password",
			setupMock: func(m *mocks.MockStore) {
				hash, hashErr := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
				require.NoError(t, hashErr)
				m.EXPECT().GetUser(gomock.Any()).Return(db.User{
					ID:           1,
					Username:     "admin",
					PasswordHash: string(hash),
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				}, nil)
			},
			expectError: true,
		},
		{
			name:     "user not found",
			username: "admin",
			password: "password",
			setupMock: func(m *mocks.MockStore) {
				m.EXPECT().GetUser(gomock.Any()).Return(db.User{}, sql.ErrNoRows)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			service, mockStore := setupTestService(ctrl)
			tt.setupMock(mockStore)

			err := service.ValidatePassword(context.Background(), tt.username, tt.password)

			if tt.expectError {
				require.Error(t, err)
				assert.True(
					t,
					errors.Is(err, ErrInvalidPassword) || errors.Is(err, ErrUserNotFound),
				)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestService_UpdateUsername(t *testing.T) {
	tests := []struct {
		name        string
		newUsername string
		setupMock   func(*mocks.MockStore)
		expectError bool
	}{
		{
			name:        "success updates username",
			newUsername: "newadmin",
			setupMock: func(m *mocks.MockStore) {
				hash, hashErr := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
				require.NoError(t, hashErr)
				m.EXPECT().UpdateUserUsername(gomock.Any(), "newadmin").Return(db.User{
					ID:           1,
					Username:     "newadmin",
					PasswordHash: string(hash),
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				}, nil)
			},
			expectError: false,
		},
		{
			name:        "database error returns error",
			newUsername: "newadmin",
			setupMock: func(m *mocks.MockStore) {
				m.EXPECT().
					UpdateUserUsername(gomock.Any(), "newadmin").
					Return(db.User{}, errors.New("database error"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			service, mockStore := setupTestService(ctrl)
			tt.setupMock(mockStore)

			err := service.UpdateUsername(context.Background(), tt.newUsername)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestService_UpdatePassword(t *testing.T) {
	tests := []struct {
		name        string
		newPassword string
		setupMock   func(*mocks.MockStore)
		expectError bool
	}{
		{
			name:        "success updates password",
			newPassword: "newpassword",
			setupMock: func(m *mocks.MockStore) {
				m.EXPECT().UpdateUserPassword(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, passwordHash string) (db.User, error) {
						// Verify the hash is valid
						err := bcrypt.CompareHashAndPassword(
							[]byte(passwordHash),
							[]byte("newpassword"),
						)
						if err != nil {
							return db.User{}, err
						}
						return db.User{
							ID:           1,
							Username:     "admin",
							PasswordHash: passwordHash,
							CreatedAt:    time.Now(),
							UpdatedAt:    time.Now(),
						}, nil
					})
			},
			expectError: false,
		},
		{
			name:        "database error returns error",
			newPassword: "newpassword",
			setupMock: func(m *mocks.MockStore) {
				m.EXPECT().
					UpdateUserPassword(gomock.Any(), gomock.Any()).
					Return(db.User{}, errors.New("database error"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			service, mockStore := setupTestService(ctrl)
			tt.setupMock(mockStore)

			err := service.UpdatePassword(context.Background(), tt.newPassword)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestService_UpdateCredentials(t *testing.T) {
	tests := []struct {
		name        string
		newUsername string
		newPassword string
		setupMock   func(*mocks.MockStore)
		expectError bool
	}{
		{
			name:        "success updates both username and password",
			newUsername: "newadmin",
			newPassword: "newpassword",
			setupMock: func(m *mocks.MockStore) {
				m.EXPECT().UpdateUserCredentials(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, arg db.UpdateUserCredentialsParams) (db.User, error) {
						// Verify the hash is valid
						err := bcrypt.CompareHashAndPassword(
							[]byte(arg.PasswordHash),
							[]byte("newpassword"),
						)
						if err != nil {
							return db.User{}, err
						}
						assert.Equal(t, "newadmin", arg.Username)
						return db.User{
							ID:           1,
							Username:     arg.Username,
							PasswordHash: arg.PasswordHash,
							CreatedAt:    time.Now(),
							UpdatedAt:    time.Now(),
						}, nil
					})
			},
			expectError: false,
		},
		{
			name:        "database error returns error",
			newUsername: "newadmin",
			newPassword: "newpassword",
			setupMock: func(m *mocks.MockStore) {
				m.EXPECT().
					UpdateUserCredentials(gomock.Any(), gomock.Any()).
					Return(db.User{}, errors.New("database error"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			service, mockStore := setupTestService(ctrl)
			tt.setupMock(mockStore)

			err := service.UpdateCredentials(context.Background(), tt.newUsername, tt.newPassword)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestService_InitializeDefaultUser(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(*mocks.MockStore)
		expectError bool
	}{
		{
			name: "user exists does nothing",
			setupMock: func(m *mocks.MockStore) {
				hash, hashErr := bcrypt.GenerateFromPassword([]byte("octobud"), bcrypt.DefaultCost)
				require.NoError(t, hashErr)
				m.EXPECT().GetUser(gomock.Any()).Return(db.User{
					ID:           1,
					Username:     "octobud",
					PasswordHash: string(hash),
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				}, nil)
			},
			expectError: false,
		},
		{
			name: "user does not exist creates default user",
			setupMock: func(m *mocks.MockStore) {
				m.EXPECT().GetUser(gomock.Any()).Return(db.User{}, sql.ErrNoRows)
				m.EXPECT().CreateUser(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, arg db.CreateUserParams) (db.User, error) {
						// Verify the hash is valid for "octobud" (default password)
						err := bcrypt.CompareHashAndPassword(
							[]byte(arg.PasswordHash),
							[]byte("octobud"),
						)
						if err != nil {
							return db.User{}, err
						}
						assert.Equal(t, "octobud", arg.Username)
						return db.User{
							ID:           1,
							Username:     arg.Username,
							PasswordHash: arg.PasswordHash,
							CreatedAt:    time.Now(),
							UpdatedAt:    time.Now(),
						}, nil
					})
			},
			expectError: false,
		},
		{
			name: "database error on get user returns error",
			setupMock: func(m *mocks.MockStore) {
				m.EXPECT().GetUser(gomock.Any()).Return(db.User{}, errors.New("database error"))
			},
			expectError: true,
		},
		{
			name: "database error on create user returns error",
			setupMock: func(m *mocks.MockStore) {
				m.EXPECT().GetUser(gomock.Any()).Return(db.User{}, sql.ErrNoRows)
				m.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(db.User{}, errors.New("database error"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			service, mockStore := setupTestService(ctrl)
			tt.setupMock(mockStore)

			err := service.InitializeDefaultUser(context.Background())

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
