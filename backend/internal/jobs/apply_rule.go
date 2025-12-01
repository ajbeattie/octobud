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

// Package jobs provides the job service.
package jobs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/riverqueue/river"

	"github.com/ajbeattie/octobud/backend/internal/db"
	"github.com/ajbeattie/octobud/backend/internal/models"
	"github.com/ajbeattie/octobud/backend/internal/query"
)

// ApplyRuleArgs represents a rule to apply retroactively to existing notifications
type ApplyRuleArgs struct {
	RuleID int64 `json:"rule_id"`
}

// Kind specifies the job type.
func (ApplyRuleArgs) Kind() string { return "apply_rule" }

// InsertOpts specifies the queue or other options to use for the job.
func (ApplyRuleArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue: "apply_rule",
	}
}

// ApplyRuleWorker handles retroactive application of rules to existing notifications
type ApplyRuleWorker struct {
	river.WorkerDefaults[ApplyRuleArgs]
	store   db.Store
	matcher RuleMatcherInterface
}

// NewApplyRuleWorker creates a new ApplyRuleWorker.
func NewApplyRuleWorker(store db.Store) *ApplyRuleWorker {
	return &ApplyRuleWorker{
		store:   store,
		matcher: NewRuleMatcher(store),
	}
}

// NewApplyRuleWorkerWithMatcher creates a worker with a custom rule matcher (useful for testing)
func NewApplyRuleWorkerWithMatcher(store db.Store, matcher RuleMatcherInterface) *ApplyRuleWorker {
	return &ApplyRuleWorker{
		store:   store,
		matcher: matcher,
	}
}

// Work applies a rule to a notification.
func (w *ApplyRuleWorker) Work(ctx context.Context, job *river.Job[ApplyRuleArgs]) error {
	ruleID := job.Args.RuleID

	// Fetch the rule
	rule, err := w.store.GetRule(ctx, ruleID)
	if err != nil {
		return fmt.Errorf("failed to get rule %d: %w", ruleID, err)
	}

	// Skip applying if rule is disabled
	if !rule.Enabled {
		return nil
	}

	// Determine the query to use - prefer viewId if both are defined
	var queryStr string

	if rule.ViewID.Valid {
		// Rule is linked to a view - resolve the view's query dynamically
		view, err := w.store.GetView(ctx, rule.ViewID.Int64)
		if err != nil {
			return fmt.Errorf("failed to get view for rule: %w", err)
		}

		if !view.Query.Valid || view.Query.String == "" {
			return fmt.Errorf("view %d has no query defined", view.ID)
		}

		queryStr = view.Query.String
	} else {
		// Rule has its own query (only use if viewId is not set)
		// Query is now nullable after migration 000024
		if !rule.Query.Valid || rule.Query.String == "" {
			return fmt.Errorf("rule %d has neither query nor viewId set", ruleID)
		}
		queryStr = rule.Query.String
	}

	// Build query from rule query string, using in:anywhere to include all notifications
	// Combine with in:anywhere to ensure we include filtered/archived/etc notifications
	fullQueryStr := fmt.Sprintf("(%s) AND in:anywhere", queryStr)

	// Build the query - use a reasonable page size for batch processing
	const pageSize = 100
	offset := int32(0)
	totalProcessed := 0
	totalMatched := 0

	for {
		dbQuery, err := query.BuildQuery(fullQueryStr, pageSize, offset)
		if err != nil {
			return fmt.Errorf("failed to build query: %w", err)
		}

		// Execute query to get matching notifications
		result, err := w.store.ListNotificationsFromQuery(ctx, dbQuery)
		if err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}

		if len(result.Notifications) == 0 {
			break // No more notifications to process
		}

		// Apply rule actions to each notification
		for _, notification := range result.Notifications {
			// Parse and apply actions
			var actions models.RuleActions
			if len(rule.Actions) > 0 {
				if err := json.Unmarshal(rule.Actions, &actions); err != nil {
					continue
				}
			}

			if err := w.matcher.ApplyRuleActions(ctx, notification.GithubID, actions); err != nil {
				// Continue processing other notifications even if one fails
				continue
			}
			totalMatched++
			totalProcessed++
		}

		// Move to next page
		offset += pageSize

		// Safety check to avoid infinite loops (shouldn't happen, but be safe)
		if totalProcessed >= int(result.Total) {
			break
		}
	}

	return nil
}
