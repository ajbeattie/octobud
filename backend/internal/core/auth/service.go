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

// Package auth provides the auth service.
package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/sqlc-dev/pqtype"
	"golang.org/x/crypto/bcrypt"

	"github.com/ajbeattie/octobud/backend/internal/db"
	"github.com/ajbeattie/octobud/backend/internal/models"
)

// Error definitions
var (
	ErrUserNotFound      = errors.New("user not found")
	ErrInvalidPassword   = errors.New("invalid password")
	ErrUserAlreadyExists = errors.New("user already exists")
)

//go:generate mockgen -source=service.go -destination=mocks/mock_service.go -package=mocks

// AuthService is the interface for the auth service.
//
//nolint:revive // exported type name stutters with package name
type AuthService interface {
	GetUser(ctx context.Context) (*models.User, error)
	ValidatePassword(ctx context.Context, username, password string) error
	UpdateUsername(ctx context.Context, newUsername string) error
	UpdatePassword(ctx context.Context, newPassword string) error
	UpdateCredentials(ctx context.Context, newUsername, newPassword string) error
	InitializeDefaultUser(ctx context.Context) error
	GetUserSyncSettings(ctx context.Context) (*models.SyncSettings, error)
	UpdateUserSyncSettings(ctx context.Context, settings *models.SyncSettings) error
	HasSyncSettings(ctx context.Context) (bool, error)
}

// Service provides business logic for authentication operations
type Service struct {
	queries db.Store
}

// NewService constructs a Service backed by the provided queries
func NewService(queries db.Store) *Service {
	return &Service{
		queries: queries,
	}
}

// GetUser retrieves the single user from the database
func (s *Service) GetUser(ctx context.Context) (*models.User, error) {
	user, err := s.queries.GetUser(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	syncSettings, err := models.SyncSettingsFromJSON(user.SyncSettings.RawMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to parse sync settings: %w", err)
	}

	return &models.User{
		ID:           user.ID,
		Username:     user.Username,
		PasswordHash: user.PasswordHash,
		SyncSettings: syncSettings,
	}, nil
}

// ValidatePassword validates the given username and password against the stored user
// Always performs bcrypt comparison to prevent timing attacks that could leak username existence
func (s *Service) ValidatePassword(ctx context.Context, username, password string) error {
	user, err := s.GetUser(ctx)

	// Use a dummy hash if user not found to prevent timing attacks
	// This ensures bcrypt comparison always takes similar time regardless of user existence
	var hashToCompare string
	var actualUsername string

	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			// Generate a dummy hash to compare against (same cost as real hashes)
			// This ensures timing is consistent whether user exists or not
			// We must generate a valid bcrypt hash, so we always generate one
			dummyHash, dummyErr := bcrypt.GenerateFromPassword([]byte("dummy"), bcrypt.DefaultCost)
			if dummyErr != nil {
				// If we can't generate dummy hash, return the error
				// This should never happen in practice, but handle it gracefully
				return fmt.Errorf("failed to generate dummy hash: %w", dummyErr)
			}
			hashToCompare = string(dummyHash)
			actualUsername = "" // No user exists
		} else {
			// Some other error occurred
			return err
		}
	} else {
		hashToCompare = user.PasswordHash
		actualUsername = user.Username
	}

	// Always perform bcrypt comparison to normalize timing
	err = bcrypt.CompareHashAndPassword([]byte(hashToCompare), []byte(password))

	// Check both password and username after comparison
	// Return ErrInvalidPassword for both invalid username and invalid password
	// This prevents leaking information about username existence
	if err != nil || actualUsername != username {
		return ErrInvalidPassword
	}

	return nil
}

// UpdateUsername updates the username in the database
func (s *Service) UpdateUsername(ctx context.Context, newUsername string) error {
	if err := ValidateUsername(newUsername); err != nil {
		return err
	}

	_, err := s.queries.UpdateUserUsername(ctx, newUsername)
	if err != nil {
		return fmt.Errorf("failed to update username: %w", err)
	}
	return nil
}

// UpdatePassword hashes and updates the password in the database
func (s *Service) UpdatePassword(ctx context.Context, newPassword string) error {
	if err := ValidatePasswordStrength(newPassword); err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	_, err = s.queries.UpdateUserPassword(ctx, string(hash))
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}
	return nil
}

// UpdateCredentials updates both username and password in the database
func (s *Service) UpdateCredentials(ctx context.Context, newUsername, newPassword string) error {
	if err := ValidateUsername(newUsername); err != nil {
		return err
	}

	if err := ValidatePasswordStrength(newPassword); err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	_, err = s.queries.UpdateUserCredentials(ctx, db.UpdateUserCredentialsParams{
		Username:     newUsername,
		PasswordHash: string(hash),
	})
	if err != nil {
		return fmt.Errorf("failed to update credentials: %w", err)
	}
	return nil
}

// InitializeDefaultUser creates the default octobud:octobud user if no user exists
func (s *Service) InitializeDefaultUser(ctx context.Context) error {
	_, err := s.GetUser(ctx)
	if err == nil {
		// User already exists
		return nil
	}

	if !errors.Is(err, ErrUserNotFound) {
		// Some other error occurred
		return fmt.Errorf("failed to check for existing user: %w", err)
	}

	// User doesn't exist, create default octobud:octobud
	defaultPassword := "octobud"
	hash, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash default password: %w", err)
	}

	_, err = s.queries.CreateUser(ctx, db.CreateUserParams{
		Username:     "octobud",
		PasswordHash: string(hash),
	})
	if err != nil {
		return fmt.Errorf("failed to create default user: %w", err)
	}

	return nil
}

// GetUserSyncSettings retrieves the user's sync settings
func (s *Service) GetUserSyncSettings(ctx context.Context) (*models.SyncSettings, error) {
	user, err := s.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	return user.SyncSettings, nil
}

// UpdateUserSyncSettings updates the user's sync settings
func (s *Service) UpdateUserSyncSettings(ctx context.Context, settings *models.SyncSettings) error {
	jsonData, err := settings.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal sync settings: %w", err)
	}

	var rawMessage pqtype.NullRawMessage
	if len(jsonData) > 0 {
		rawMessage = pqtype.NullRawMessage{
			RawMessage: jsonData,
			Valid:      true,
		}
	}

	_, err = s.queries.UpdateUserSyncSettings(ctx, rawMessage)
	if err != nil {
		return fmt.Errorf("failed to update sync settings: %w", err)
	}
	return nil
}

// HasSyncSettings checks if the user has sync settings configured
func (s *Service) HasSyncSettings(ctx context.Context) (bool, error) {
	settings, err := s.GetUserSyncSettings(ctx)
	if err != nil {
		return false, err
	}
	return settings != nil && settings.SetupCompleted, nil
}
