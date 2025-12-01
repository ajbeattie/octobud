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

package sql

import (
	"testing"

	"github.com/ajbeattie/octobud/backend/internal/db"
	"github.com/ajbeattie/octobud/backend/internal/query/parse"
)

func TestBuilder_BasicTerms(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantWhere string
		wantArgs  []interface{}
		wantJoins int
	}{
		{
			name:      "simple repo term",
			input:     "repo:cli",
			wantWhere: "r.full_name ILIKE $1",
			wantArgs:  []interface{}{"%cli%"},
			wantJoins: 1,
		},
		{
			name:      "reason term",
			input:     "reason:review_requested",
			wantWhere: "n.reason ILIKE $1",
			wantArgs:  []interface{}{"%review_requested%"},
			wantJoins: 0,
		},
		{
			name:      "type term",
			input:     "type:Issue",
			wantWhere: "n.subject_type ILIKE $1",
			wantArgs:  []interface{}{"%Issue%"},
			wantJoins: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast, err := parseQuery(tt.input)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			builder := NewBuilder()
			var query db.NotificationQuery
			query, err = builder.Build(ast)
			if err != nil {
				t.Fatalf("build error: %v", err)
			}

			// Check WHERE clause contains expected condition
			if len(query.Where) == 0 {
				t.Fatal("expected non-empty WHERE clause")
			}

			whereClause := query.Where[0]
			if !contains(whereClause, tt.wantWhere) {
				t.Errorf("expected WHERE to contain %q, got %q", tt.wantWhere, whereClause)
			}

			// Check args
			if len(query.Args) != len(tt.wantArgs) {
				t.Errorf("expected %d args, got %d", len(tt.wantArgs), len(query.Args))
			}

			// Check joins
			if len(query.Joins) != tt.wantJoins {
				t.Errorf("expected %d joins, got %d", tt.wantJoins, len(query.Joins))
			}
		})
	}
}

func TestBuilder_LogicalOperators(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantParts []string // Parts that should appear in the WHERE clause
	}{
		{
			name:      "AND operator",
			input:     "repo:cli AND is:unread",
			wantParts: []string{"r.full_name", "n.is_read = FALSE", "AND"},
		},
		{
			name:      "OR operator",
			input:     "repo:cli OR repo:other",
			wantParts: []string{"r.full_name", "$1", "$2", "OR"},
		},
		{
			name:      "NOT operator",
			input:     "NOT repo:cli",
			wantParts: []string{"NOT", "r.full_name"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast, err := parseQuery(tt.input)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			builder := NewBuilder()
			query, err := builder.Build(ast)
			if err != nil {
				t.Fatalf("build error: %v", err)
			}

			if len(query.Where) == 0 {
				t.Fatal("expected non-empty WHERE clause")
			}

			whereClause := query.Where[0]
			for _, part := range tt.wantParts {
				if !contains(whereClause, part) {
					t.Errorf("expected WHERE to contain %q, got %q", part, whereClause)
				}
			}
		})
	}
}

func TestBuilder_InOperator(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantContains []string
	}{
		{
			name:         "in:inbox",
			input:        "in:inbox",
			wantContains: []string{"n.archived = FALSE", "snoozed_until", "n.muted = FALSE"},
		},
		{
			name:         "in:archive",
			input:        "in:archive",
			wantContains: []string{"n.archived = TRUE", "n.muted = FALSE"},
		},
		{
			name:         "in:snoozed",
			input:        "in:snoozed",
			wantContains: []string{"n.snoozed_until IS NOT NULL", "NOW()", "n.archived = FALSE"},
		},
		{
			name:  "in:filtered",
			input: "in:filtered",
			wantContains: []string{
				"n.filtered = TRUE",
				"n.archived = FALSE",
				"snoozed_until",
				"n.muted = FALSE",
			},
		},
		{
			name:         "in:anywhere",
			input:        "in:anywhere",
			wantContains: []string{"TRUE"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast, err := parseQuery(tt.input)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			builder := NewBuilder()
			query, err := builder.Build(ast)
			if err != nil {
				t.Fatalf("build error: %v", err)
			}

			if len(query.Where) == 0 {
				t.Fatal("expected non-empty WHERE clause")
			}

			whereClause := query.Where[0]
			for _, part := range tt.wantContains {
				if !contains(whereClause, part) {
					t.Errorf("expected WHERE to contain %q, got %q", part, whereClause)
				}
			}
		})
	}
}

func TestBuilder_IsOperator(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantWhere string
	}{
		{
			name:      "is:unread",
			input:     "is:unread",
			wantWhere: "n.is_read = FALSE",
		},
		{
			name:      "is:read",
			input:     "is:read",
			wantWhere: "n.is_read = TRUE",
		},
		{
			name:      "is:archived",
			input:     "is:archived",
			wantWhere: "n.archived = TRUE",
		},
		{
			name:      "is:muted",
			input:     "is:muted",
			wantWhere: "n.muted = TRUE",
		},
		{
			name:      "is:snoozed",
			input:     "is:snoozed",
			wantWhere: "n.snoozed_until IS NOT NULL AND n.snoozed_until > NOW()",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast, err := parseQuery(tt.input)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			builder := NewBuilder()
			query, err := builder.Build(ast)
			if err != nil {
				t.Fatalf("build error: %v", err)
			}

			if len(query.Where) == 0 {
				t.Fatal("expected non-empty WHERE clause")
			}

			whereClause := query.Where[0]
			if !contains(whereClause, tt.wantWhere) {
				t.Errorf("expected WHERE to contain %q, got %q", tt.wantWhere, whereClause)
			}
		})
	}
}

func TestBuilder_CommaOR(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantParts []string
	}{
		{
			name:      "comma separated repos",
			input:     "repo:cli,other",
			wantParts: []string{"r.full_name", "$1", "$2", "OR"},
		},
		{
			name:      "comma separated reasons",
			input:     "reason:review_requested,mention",
			wantParts: []string{"n.reason", "$1", "$2", "OR"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast, err := parseQuery(tt.input)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			builder := NewBuilder()
			query, err := builder.Build(ast)
			if err != nil {
				t.Fatalf("build error: %v", err)
			}

			if len(query.Where) == 0 {
				t.Fatal("expected non-empty WHERE clause")
			}

			whereClause := query.Where[0]
			for _, part := range tt.wantParts {
				if !contains(whereClause, part) {
					t.Errorf("expected WHERE to contain %q, got %q", part, whereClause)
				}
			}
		})
	}
}

func TestBuilder_ComplexQueries(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "grouped OR with AND",
			input: "(repo:cli OR repo:other) AND is:unread",
		},
		{
			name:  "NOT with grouping",
			input: "NOT (in:archive OR in:snoozed)",
		},
		{
			name:  "deeply nested",
			input: "((repo:cli AND is:unread) OR (in:snoozed AND repo:other)) AND NOT author:bot",
		},
		{
			name:  "mixed with free text",
			input: "repo:cli urgent fix",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast, err := parseQuery(tt.input)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			builder := NewBuilder()
			query, err := builder.Build(ast)
			if err != nil {
				t.Fatalf("build error: %v", err)
			}

			// Just verify we can build complex queries without errors
			if len(query.Where) == 0 {
				t.Fatal("expected non-empty WHERE clause")
			}
		})
	}
}

// Note: Tests for default filters (BuildQuery) are in the main query package
// to avoid circular dependencies. This package only tests the pure SQL builder.

func TestBuilder_Joins(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantJoins []string
	}{
		{
			name:      "repo field requires repo join",
			input:     "repo:cli",
			wantJoins: []string{"repositories"},
		},
		{
			name:      "org field requires repo join",
			input:     "org:github",
			wantJoins: []string{"repositories"},
		},
		{
			name:      "reason doesn't require join",
			input:     "reason:mention",
			wantJoins: []string{},
		},
		{
			name:      "free text requires repo join",
			input:     "urgent",
			wantJoins: []string{"repositories"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast, err := parseQuery(tt.input)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			builder := NewBuilder()
			query, err := builder.Build(ast)
			if err != nil {
				t.Fatalf("build error: %v", err)
			}

			for _, expectedJoin := range tt.wantJoins {
				found := false
				for _, join := range query.Joins {
					if contains(join, expectedJoin) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected join containing %q, got %v", expectedJoin, query.Joins)
				}
			}

			if len(tt.wantJoins) == 0 && len(query.Joins) > 0 {
				t.Errorf("expected no joins, got %v", query.Joins)
			}
		})
	}
}

// Helper functions

func parseQuery(input string) (parse.Node, error) {
	if input == "" {
		return nil, nil
	}

	lexer := parse.NewLexer(input)
	tokens, err := lexer.Tokenize()
	if err != nil {
		return nil, err
	}

	parser := parse.NewParser(tokens)
	return parser.Parse()
}

func contains(s, substr string) bool {
	return s != "" && substr != "" && (s == substr || containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
