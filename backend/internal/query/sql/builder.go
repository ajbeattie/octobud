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

// Package sql provides the SQL builder for the application.
package sql

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ajbeattie/octobud/backend/internal/db"
	"github.com/ajbeattie/octobud/backend/internal/query/parse"
)

const (
	queryValueSnoozed  = "snoozed"
	queryValueFiltered = "filtered"
	queryValueTrue     = "true"
	queryValueFalse    = "false"
	queryValueYes      = "yes"
)

// Error definitions
var (
	ErrUnknownNodeType        = errors.New("unknown node type")
	ErrUnsupportedField       = errors.New("unsupported field")
	ErrInvalidInOperatorValue = errors.New("invalid value for in: operator")
	ErrInvalidIsOperatorValue = errors.New("invalid value for is: operator")
	ErrInvalidBooleanValue    = errors.New("invalid boolean value")
	ErrInvalidSnoozedValue    = errors.New("invalid boolean value for snoozed")
	ErrInvalidMergedValue     = errors.New("invalid value for merged field")
	ErrTagsFieldRequiresValue = errors.New("tags field requires at least one value")
)

// Builder builds SQL queries from AST nodes
type Builder struct {
	joins      map[string]bool
	args       []interface{}
	argCounter int
}

// NewBuilder creates a new SQL builder
func NewBuilder() *Builder {
	return &Builder{
		joins:      make(map[string]bool),
		args:       []interface{}{},
		argCounter: 0,
	}
}

// Build generates a NotificationQuery from an AST node
// This is a pure translation - no default filters are applied.
// The caller is responsible for applying any business logic defaults.
func (b *Builder) Build(node parse.Node) (db.NotificationQuery, error) {
	if node == nil {
		// Empty query - return empty result
		return db.NotificationQuery{
			Joins:  []string{},
			Where:  []string{},
			Args:   []interface{}{},
			Limit:  0,
			Offset: 0,
		}, nil
	}

	whereExpr, err := b.visitNode(node)
	if err != nil {
		return db.NotificationQuery{}, err
	}

	// Convert joins map to slice
	joins := make([]string, 0, len(b.joins))
	for join := range b.joins {
		joins = append(joins, join)
	}

	var where []string
	if whereExpr != "" {
		where = []string{whereExpr}
	}

	return db.NotificationQuery{
		Joins:  joins,
		Where:  where,
		Args:   b.args,
		Limit:  0,
		Offset: 0,
	}, nil
}

// visitNode visits a node and generates SQL
func (b *Builder) visitNode(node parse.Node) (string, error) {
	switch n := node.(type) {
	case *parse.BinaryExpr:
		return b.visitBinaryExpr(n)
	case *parse.NotExpr:
		return b.visitNotExpr(n)
	case *parse.Term:
		return b.visitTerm(n)
	case *parse.FreeText:
		return b.visitFreeText(n)
	case *parse.ParenExpr:
		return b.visitNode(n.Expr)
	default:
		return "", errors.Join(ErrUnknownNodeType, fmt.Errorf("node type: %T", node))
	}
}

// visitBinaryExpr handles AND/OR expressions
func (b *Builder) visitBinaryExpr(node *parse.BinaryExpr) (string, error) {
	left, err := b.visitNode(node.Left)
	if err != nil {
		return "", err
	}

	right, err := b.visitNode(node.Right)
	if err != nil {
		return "", err
	}

	op := node.Op
	return fmt.Sprintf("(%s %s %s)", left, op, right), nil
}

// visitNotExpr handles NOT expressions
func (b *Builder) visitNotExpr(node *parse.NotExpr) (string, error) {
	expr, err := b.visitNode(node.Expr)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("NOT (%s)", expr), nil
}

// visitTerm handles field:value terms
func (b *Builder) visitTerm(node *parse.Term) (string, error) {
	field := strings.ToLower(strings.TrimSpace(node.Field))

	// Handle special operators
	switch field {
	case "in":
		return b.handleInOperator(node.Values)
	case "is":
		return b.handleIsOperator(node.Values)
	case "repo", "repository":
		return b.handleRepoField(node.Values)
	case "org":
		return b.handleOrgField(node.Values)
	case "reason":
		return b.handleReasonField(node.Values)
	case "type", "subject_type":
		return b.handleTypeField(node.Values)
	case "author":
		return b.handleAuthorField(node.Values)
	case "state":
		return b.handleStateField(node.Values)
	case "merged":
		return b.handleMergedField(node.Values)
	case "state_reason":
		return b.handleStateReasonField(node.Values)
	case "read":
		return b.handleReadField(node.Values)
	case "archived":
		return b.handleArchivedField(node.Values)
	case "muted":
		return b.handleMutedField(node.Values)
	case queryValueSnoozed:
		return b.handleSnoozedField(node.Values)
	case queryValueFiltered:
		return b.handleFilteredField(node.Values)
	case "tags":
		return b.handleTagsField(node.Values)
	default:
		return "", errors.Join(ErrUnsupportedField, fmt.Errorf("field: %s", field))
	}
}

// visitFreeText handles free text search
func (b *Builder) visitFreeText(node *parse.FreeText) (string, error) {
	b.requireRepoJoin()
	pattern := "%" + node.Text + "%"

	placeholder1 := b.addArg(pattern)
	placeholder2 := b.addArg(pattern)
	placeholder3 := b.addArg(pattern)
	placeholder4 := b.addArg(pattern)
	placeholder5 := b.addArg(pattern)
	placeholder6 := b.addArg(pattern)

	return fmt.Sprintf(
		"(n.subject_title ILIKE %s OR n.subject_type ILIKE %s OR r.full_name ILIKE %s OR "+
			"n.author_login ILIKE %s OR n.subject_state ILIKE %s OR n.subject_number::text ILIKE %s)",
		placeholder1,
		placeholder2,
		placeholder3,
		placeholder4,
		placeholder5,
		placeholder6,
	), nil
}

// Field handler methods

func (b *Builder) handleInOperator(values []string) (string, error) {
	// in: operator controls lifecycle filtering
	// in:inbox - exclude archived, snoozed, muted
	// in:archive - show only archived (exclude muted)
	// in:snoozed - show only snoozed (exclude archived, muted)
	// in:anywhere - show all (no lifecycle filters)

	var conditions []string
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		switch value {
		case "inbox":
			conditions = append(
				conditions,
				"(n.archived = FALSE AND (n.snoozed_until IS NULL OR n.snoozed_until <= NOW()) "+
					"AND n.muted = FALSE AND n.filtered = FALSE)",
			)
		case "archive":
			conditions = append(conditions, "(n.archived = TRUE AND n.muted = FALSE)")
		case queryValueSnoozed:
			conditions = append(
				conditions,
				"(n.snoozed_until IS NOT NULL AND n.snoozed_until > NOW() AND n.archived = FALSE AND n.muted = FALSE)",
			)
		case queryValueFiltered:
			conditions = append(conditions, "(n.filtered = TRUE)")
		case "anywhere":
			// No filter - show all
			conditions = append(conditions, "TRUE")
		default:
			return "", errors.Join(ErrInvalidInOperatorValue, fmt.Errorf("value: %s", value))
		}
	}

	if len(conditions) == 1 {
		return conditions[0], nil
	}
	return "(" + strings.Join(conditions, " OR ") + ")", nil
}

func (b *Builder) handleIsOperator(values []string) (string, error) {
	// is: operator is an alias for common filters
	var conditions []string
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		switch value {
		case "unread":
			conditions = append(conditions, "n.is_read = FALSE")
		case "read":
			conditions = append(conditions, "n.is_read = TRUE")
		case "archived":
			conditions = append(conditions, "n.archived = TRUE")
		case "muted":
			conditions = append(conditions, "n.muted = TRUE")
		case queryValueSnoozed:
			conditions = append(
				conditions,
				"(n.snoozed_until IS NOT NULL AND n.snoozed_until > NOW())",
			)
		case "starred":
			conditions = append(conditions, "n.starred = TRUE")
		case queryValueFiltered:
			conditions = append(conditions, "n.filtered = TRUE")
		default:
			return "", errors.Join(ErrInvalidIsOperatorValue, fmt.Errorf("value: %s", value))
		}
	}

	if len(conditions) == 1 {
		return conditions[0], nil
	}
	return "(" + strings.Join(conditions, " OR ") + ")", nil
}

func (b *Builder) handleRepoField(values []string) (string, error) {
	b.requireRepoJoin()
	return b.buildStringFilter("r.full_name", values), nil
}

func (b *Builder) handleOrgField(values []string) (string, error) {
	b.requireRepoJoin()
	// Org is prefix matching: org:cli matches cli/*
	var conditions []string
	for _, value := range values {
		pattern := value + "/%"
		placeholder := b.addArg(pattern)
		conditions = append(conditions, fmt.Sprintf("r.full_name ILIKE %s", placeholder))
	}

	if len(conditions) == 1 {
		return conditions[0], nil
	}
	return "(" + strings.Join(conditions, " OR ") + ")", nil
}

func (b *Builder) handleReasonField(values []string) (string, error) {
	return b.buildStringFilter("n.reason", values), nil
}

func (b *Builder) handleTypeField(values []string) (string, error) {
	return b.buildStringFilter("n.subject_type", values), nil
}

func (b *Builder) handleAuthorField(values []string) (string, error) {
	return b.buildStringFilter("n.author_login", values), nil
}

func (b *Builder) handleStateField(values []string) (string, error) {
	// State is stored in subject_state column (extracted from subject_raw)
	// We use the column instead of subject_raw->>'state' for performance
	// since the JSON path index has been removed in favor of the column index
	var conditions []string
	for _, value := range values {
		placeholder := b.addArg(value)
		conditions = append(conditions, fmt.Sprintf("n.subject_state = %s", placeholder))
	}

	if len(conditions) == 1 {
		return conditions[0], nil
	}
	return "(" + strings.Join(conditions, " OR ") + ")", nil
}

func (b *Builder) handleMergedField(values []string) (string, error) {
	// Merged is stored in subject_merged column (extracted from subject_raw)
	// Only applies to Pull Requests
	var conditions []string
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		var boolVal bool
		switch value {
		case queryValueTrue, queryValueYes, "1", "merged":
			boolVal = true
		case queryValueFalse, "no", "0", "unmerged":
			boolVal = false
		default:
			return "", errors.Join(ErrInvalidMergedValue, fmt.Errorf("value: %s", value))
		}
		placeholder := b.addArg(boolVal)
		conditions = append(conditions, fmt.Sprintf("n.subject_merged = %s", placeholder))
	}

	if len(conditions) == 1 {
		return conditions[0], nil
	}
	return "(" + strings.Join(conditions, " OR ") + ")", nil
}

func (b *Builder) handleStateReasonField(values []string) (string, error) {
	// State reason is stored in subject_state_reason column (extracted from subject_raw)
	// Only applies to Issues
	return b.buildStringFilter("n.subject_state_reason", values), nil
}

func (b *Builder) handleReadField(values []string) (string, error) {
	return b.buildBooleanFilter("n.is_read", values)
}

func (b *Builder) handleArchivedField(values []string) (string, error) {
	return b.buildBooleanFilter("n.archived", values)
}

func (b *Builder) handleMutedField(values []string) (string, error) {
	return b.buildBooleanFilter("n.muted", values)
}

func (b *Builder) handleSnoozedField(values []string) (string, error) {
	var conditions []string
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		switch value {
		case "true", "yes", "1":
			conditions = append(
				conditions,
				"(n.snoozed_until IS NOT NULL AND n.snoozed_until > NOW())",
			)
		case "false", "no", "0":
			conditions = append(conditions, "(n.snoozed_until IS NULL OR n.snoozed_until <= NOW())")
		default:
			return "", errors.Join(ErrInvalidSnoozedValue, fmt.Errorf("value: %s", value))
		}
	}

	if len(conditions) == 1 {
		return conditions[0], nil
	}
	return "(" + strings.Join(conditions, " OR ") + ")", nil
}

func (b *Builder) handleFilteredField(values []string) (string, error) {
	return b.buildBooleanFilter("n.filtered", values)
}

func (b *Builder) handleTagsField(values []string) (string, error) {
	// tags:foo,bar uses OR logic - notification must have at least one of these tags
	// For AND logic, use multiple separate terms: tags:foo AND tags:bar

	if len(values) == 0 {
		return "", ErrTagsFieldRequiresValue
	}

	// Build a subquery to get tag IDs by slug pattern matching
	// Using array overlap operator &&
	// Supports partial matching: tags:urg matches "urgent", "urgent-bug", etc.
	var conditions []string
	for _, value := range values {
		pattern := "%" + value + "%"
		placeholder := b.addArg(pattern)
		conditions = append(conditions, fmt.Sprintf("slug ILIKE %s", placeholder))
	}

	// Check if notification's tag_ids array overlaps with the tag IDs for the given slug patterns
	// n.tag_ids && ARRAY(SELECT id FROM tags WHERE slug ILIKE '%pattern1%' OR slug ILIKE '%pattern2%')
	return fmt.Sprintf(
		"n.tag_ids && ARRAY(SELECT id FROM tags WHERE %s)",
		strings.Join(conditions, " OR "),
	), nil
}

// Helper methods

func (b *Builder) buildStringFilter(column string, values []string) string {
	var conditions []string
	for _, value := range values {
		pattern := "%" + value + "%"
		placeholder := b.addArg(pattern)
		conditions = append(conditions, fmt.Sprintf("%s ILIKE %s", column, placeholder))
	}

	if len(conditions) == 1 {
		return conditions[0]
	}
	return "(" + strings.Join(conditions, " OR ") + ")"
}

func (b *Builder) buildBooleanFilter(column string, values []string) (string, error) {
	var conditions []string
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		switch value {
		case "true", "yes", "1":
			conditions = append(conditions, fmt.Sprintf("%s = TRUE", column))
		case "false", "no", "0":
			conditions = append(conditions, fmt.Sprintf("%s = FALSE", column))
		default:
			return "", errors.Join(ErrInvalidBooleanValue, fmt.Errorf("value: %s", value))
		}
	}

	if len(conditions) == 1 {
		return conditions[0], nil
	}
	return "(" + strings.Join(conditions, " OR ") + ")", nil
}

func (b *Builder) addArg(arg interface{}) string {
	b.args = append(b.args, arg)
	b.argCounter++
	return fmt.Sprintf("$%d", b.argCounter)
}

func (b *Builder) requireRepoJoin() {
	b.joins["LEFT JOIN repositories r ON r.id = n.repository_id"] = true
}

// requirePRJoin is kept for potential future use
//
//nolint:unused // Reserved for future PR-related queries
func (b *Builder) requirePRJoin() {
	b.joins["LEFT JOIN pull_requests pr ON pr.id = n.pull_request_id"] = true
}

// For use in checking "now" for snoozed queries
var _ = time.Now
