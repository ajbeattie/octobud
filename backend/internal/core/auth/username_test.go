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
	"strings"
	"testing"
)

func TestValidateUsername(t *testing.T) {
	tests := []struct {
		name      string
		username  string
		wantError bool
		errorType error
	}{
		{
			name:      "valid username - alphanumeric",
			username:  "octobud",
			wantError: false,
		},
		{
			name:      "valid username - with underscore",
			username:  "octo_bud",
			wantError: false,
		},
		{
			name:      "valid username - with dot",
			username:  "octo.bud",
			wantError: false,
		},
		{
			name:      "valid username - with hyphen",
			username:  "octo-bud",
			wantError: false,
		},
		{
			name:      "valid username - email-like",
			username:  "octo@bud",
			wantError: false,
		},
		{
			name:      "valid username - email-like with domain",
			username:  "user@example.com",
			wantError: false,
		},
		{
			name:      "valid username - minimum length",
			username:  "abc",
			wantError: false,
		},
		{
			name:      "valid username - maximum length",
			username:  strings.Repeat("a", MaxUsernameLength),
			wantError: false,
		},
		{
			name:      "too short - 2 characters",
			username:  "ab",
			wantError: true,
			errorType: ErrUsernameTooShort,
		},
		{
			name:      "too short - empty",
			username:  "",
			wantError: true,
			errorType: ErrUsernameEmpty,
		},
		{
			name:      "too long - 65 characters",
			username:  strings.Repeat("a", MaxUsernameLength+1),
			wantError: true,
			errorType: ErrUsernameTooLong,
		},
		{
			name:      "invalid - starts with underscore",
			username:  "_octobud",
			wantError: true,
			errorType: ErrUsernameInvalid,
		},
		{
			name:      "invalid - starts with dot",
			username:  ".octobud",
			wantError: true,
			errorType: ErrUsernameInvalid,
		},
		{
			name:      "invalid - starts with hyphen",
			username:  "-octobud",
			wantError: true,
			errorType: ErrUsernameInvalid,
		},
		{
			name:      "invalid - starts with @",
			username:  "@octobud",
			wantError: true,
			errorType: ErrUsernameInvalid,
		},
		{
			name:      "invalid - ends with underscore",
			username:  "octobud_",
			wantError: true,
			errorType: ErrUsernameInvalid,
		},
		{
			name:      "invalid - ends with dot",
			username:  "octobud.",
			wantError: true,
			errorType: ErrUsernameInvalid,
		},
		{
			name:      "invalid - ends with hyphen",
			username:  "octobud-",
			wantError: true,
			errorType: ErrUsernameInvalid,
		},
		{
			name:      "invalid - ends with @",
			username:  "octobud@",
			wantError: true,
			errorType: ErrUsernameInvalid,
		},
		{
			name:      "invalid - special characters",
			username:  "octo#bud",
			wantError: true,
			errorType: ErrUsernameInvalid,
		},
		{
			name:      "invalid - spaces",
			username:  "octo bud",
			wantError: true,
			errorType: ErrUsernameInvalid,
		},
		{
			name:      "valid - unicode alphanumeric",
			username:  "用户123",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUsername(tt.username)

			if tt.wantError {
				if err == nil {
					t.Errorf("ValidateUsername() expected error, got nil")
					return
				}
				if !errors.Is(err, tt.errorType) {
					t.Errorf("ValidateUsername() error = %v, want %v", err, tt.errorType)
				}
			} else if err != nil {
				t.Errorf("ValidateUsername() unexpected error = %v", err)
			}
		})
	}
}
