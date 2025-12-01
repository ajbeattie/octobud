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
	"errors"
	"fmt"
	"regexp"
	"unicode/utf8"
)

// Error definitions
var (
	ErrUsernameTooShort = errors.New("username must be at least 3 characters long")
	ErrUsernameTooLong  = errors.New("username must be no more than 64 characters long")
	ErrUsernameInvalid  = errors.New(
		"username can only contain letters, numbers, underscores, dots, hyphens, and @ symbols",
	)
	ErrUsernameEmpty = errors.New("username cannot be empty")
)

const (
	// MinUsernameLength is the minimum allowed username length
	MinUsernameLength = 3
	// MaxUsernameLength is the maximum allowed username length
	MaxUsernameLength = 64
)

// Username pattern: allows alphanumeric (including Unicode), underscores, dots, hyphens, and @
// (for email-like usernames)
// Must start with a letter or number, can contain @ but not at the start or end
// Uses \pL for Unicode letters and \pN for Unicode numbers
var usernamePattern = regexp.MustCompile(
	`^[\pL\pN]([\pL\pN._-]*[\pL\pN])?$|^[\pL\pN][\pL\pN._-]*@[\pL\pN]([\pL\pN.-]*[\pL\pN])?$`,
)

// ValidateUsername validates that a username meets format requirements
func ValidateUsername(username string) error {
	length := utf8.RuneCountInString(username)

	if length == 0 {
		return ErrUsernameEmpty
	}

	if length < MinUsernameLength {
		return fmt.Errorf("%w (got %d)", ErrUsernameTooShort, length)
	}

	if length > MaxUsernameLength {
		return fmt.Errorf("%w (got %d)", ErrUsernameTooLong, length)
	}

	// Check pattern: alphanumeric, underscores, dots, hyphens, and @ (email-like)
	if !usernamePattern.MatchString(username) {
		return ErrUsernameInvalid
	}

	return nil
}
