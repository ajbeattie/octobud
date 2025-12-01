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

func TestValidatePasswordStrength(t *testing.T) {
	tests := []struct {
		name      string
		password  string
		wantError bool
		errorType error
	}{
		{
			name:      "valid password - minimum length",
			password:  "12345678",
			wantError: false,
		},
		{
			name:      "valid password - longer than minimum",
			password:  "this-is-a-valid-password-123",
			wantError: false,
		},
		{
			name:      "valid password - maximum length",
			password:  strings.Repeat("a", MaxPasswordLength),
			wantError: false,
		},
		{
			name:      "too short - 7 characters",
			password:  "1234567",
			wantError: true,
			errorType: ErrPasswordTooShort,
		},
		{
			name:      "too short - empty",
			password:  "",
			wantError: true,
			errorType: ErrPasswordTooShort,
		},
		{
			name:      "too long - 129 characters",
			password:  strings.Repeat("a", MaxPasswordLength+1),
			wantError: true,
			errorType: ErrPasswordTooLong,
		},
		{
			name:      "unicode characters - valid",
			password:  "密码123456",
			wantError: false,
		},
		{
			name:      "unicode characters - too short",
			password:  "密码123",
			wantError: true,
			errorType: ErrPasswordTooShort,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePasswordStrength(tt.password)

			if tt.wantError {
				if err == nil {
					t.Errorf("ValidatePasswordStrength() expected error, got nil")
					return
				}
				if !errors.Is(err, tt.errorType) {
					t.Errorf("ValidatePasswordStrength() error = %v, want %v", err, tt.errorType)
				}
			} else if err != nil {
				t.Errorf("ValidatePasswordStrength() unexpected error = %v", err)
			}
		})
	}
}
