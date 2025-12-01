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

// LoginRequest represents the request body for login
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents the response from login
type LoginResponse struct {
	Token     string `json:"token"`
	Username  string `json:"username"`
	CSRFToken string `json:"csrfToken"` // CSRF token for double-submit cookie pattern
}

// UserResponse represents the current user information
type UserResponse struct {
	Username string `json:"username"`
}

// UpdateCredentialsRequest represents the request to update credentials
type UpdateCredentialsRequest struct {
	CurrentPassword string  `json:"currentPassword"`
	NewUsername     *string `json:"newUsername,omitempty"`
	NewPassword     *string `json:"newPassword,omitempty"`
}
