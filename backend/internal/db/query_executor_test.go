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

package db

import (
	"testing"
)

func TestIncrementPlaceholders(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		offset   int
		expected string
	}{
		{
			name:     "no placeholders",
			input:    "SELECT * FROM notifications",
			offset:   1,
			expected: "SELECT * FROM notifications",
		},
		{
			name:     "single placeholder",
			input:    "WHERE x = $1",
			offset:   1,
			expected: "WHERE x = $2",
		},
		{
			name:     "multiple placeholders",
			input:    "WHERE x = $1 AND y = $2 AND z = $3",
			offset:   1,
			expected: "WHERE x = $2 AND y = $3 AND z = $4",
		},
		{
			name:     "zero offset",
			input:    "WHERE x = $1 AND y = $2",
			offset:   0,
			expected: "WHERE x = $1 AND y = $2",
		},
		{
			name:     "offset by 2",
			input:    "WHERE x = $1 AND y = $2",
			offset:   2,
			expected: "WHERE x = $3 AND y = $4",
		},
		{
			name:     "placeholder at start",
			input:    "$1",
			offset:   1,
			expected: "$2",
		},
		{
			name:     "placeholder at end",
			input:    "WHERE x = $1",
			offset:   5,
			expected: "WHERE x = $6",
		},
		{
			name:     "complex query with multiple placeholders",
			input:    "WHERE (r.name = $1 OR r.name = $2) AND n.archived = $3",
			offset:   1,
			expected: "WHERE (r.name = $2 OR r.name = $3) AND n.archived = $4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := incrementPlaceholders(tt.input, tt.offset)
			if result != tt.expected {
				t.Errorf("incrementPlaceholders() = %q, want %q", result, tt.expected)
			}
		})
	}
}
