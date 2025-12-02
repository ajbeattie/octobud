// Copyright (C) 2025 Austin Beattie
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A GENERAL PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package models

import "encoding/json"

// User represents a user in the system
type User struct {
	ID           int64
	Username     string
	PasswordHash string
	SyncSettings *SyncSettings
}

// SyncSettings represents the user's sync configuration
type SyncSettings struct {
	InitialSyncDays       *int `json:"initialSyncDays,omitempty"`     // Number of days to sync (null = unlimited)
	InitialSyncMaxCount   *int `json:"initialSyncMaxCount,omitempty"` // Maximum notifications (null = no limit)
	InitialSyncUnreadOnly bool `json:"initialSyncUnreadOnly"`         // Only sync unread notifications initially
	SetupCompleted        bool `json:"setupCompleted"`                // Whether setup was completed
}

// ToJSON converts SyncSettings to JSON bytes
func (s *SyncSettings) ToJSON() (json.RawMessage, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

// SyncSettingsFromJSON creates SyncSettings from JSON bytes
func SyncSettingsFromJSON(data json.RawMessage) (*SyncSettings, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var settings SyncSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, err
	}
	return &settings, nil
}
