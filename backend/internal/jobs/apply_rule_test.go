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

package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ajbeattie/octobud/backend/internal/db"
	dbmocks "github.com/ajbeattie/octobud/backend/internal/db/mocks"
	jobmocks "github.com/ajbeattie/octobud/backend/internal/jobs/mocks"
)

func setupTestWorker(
	ctrl *gomock.Controller,
) (*ApplyRuleWorker, *dbmocks.MockStore, *jobmocks.MockRuleMatcherInterface) {
	mockStore := dbmocks.NewMockStore(ctrl)
	mockMatcher := jobmocks.NewMockRuleMatcherInterface(ctrl)
	worker := NewApplyRuleWorkerWithMatcher(mockStore, mockMatcher)
	return worker, mockStore, mockMatcher
}

func TestApplyRuleWorker_SuccessWithRuleQuery(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	worker, mockStore, mockMatcher := setupTestWorker(ctrl)

	ruleID := int64(1)
	rule := db.Rule{
		ID:      ruleID,
		Name:    "Test Rule",
		Enabled: true,
		Query:   sql.NullString{String: "is:unread", Valid: true},
		ViewID:  sql.NullInt64{Valid: false},
		Actions: json.RawMessage(`{"markRead": true}`),
	}

	notifications := []db.Notification{
		{
			ID:       1,
			GithubID: "notif-1",
		},
		{
			ID:       2,
			GithubID: "notif-2",
		},
	}

	queryResult := db.ListNotificationsFromQueryResult{
		Notifications: notifications,
		Total:         2,
	}

	job := &river.Job[ApplyRuleArgs]{
		JobRow: &rivertype.JobRow{ID: 1},
		Args: ApplyRuleArgs{
			RuleID: ruleID,
		},
	}

	// Expect GetRule call
	mockStore.EXPECT().
		GetRule(gomock.Any(), ruleID).
		Return(rule, nil)

	// Expect ListNotificationsFromQuery call (with pagination)
	mockStore.EXPECT().
		ListNotificationsFromQuery(gomock.Any(), gomock.Any()).
		Return(queryResult, nil)

	// Expect ApplyRuleActions for each notification
	mockMatcher.EXPECT().
		ApplyRuleActions(gomock.Any(), "notif-1", gomock.Any()).
		Return(nil)
	mockMatcher.EXPECT().
		ApplyRuleActions(gomock.Any(), "notif-2", gomock.Any()).
		Return(nil)

	err := worker.Work(context.Background(), job)
	require.NoError(t, err)
}

func TestApplyRuleWorker_SuccessWithViewQuery(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	worker, mockStore, mockMatcher := setupTestWorker(ctrl)

	ruleID := int64(1)
	viewID := int64(10)
	rule := db.Rule{
		ID:      ruleID,
		Name:    "Test Rule",
		Enabled: true,
		Query:   sql.NullString{Valid: false},
		ViewID:  sql.NullInt64{Int64: viewID, Valid: true},
		Actions: json.RawMessage(`{"archive": true}`),
	}

	view := db.View{
		ID:    viewID,
		Name:  "Test View",
		Query: sql.NullString{String: "is:unread", Valid: true},
	}

	notifications := []db.Notification{
		{
			ID:       1,
			GithubID: "notif-1",
		},
	}

	queryResult := db.ListNotificationsFromQueryResult{
		Notifications: notifications,
		Total:         1,
	}

	job := &river.Job[ApplyRuleArgs]{
		JobRow: &rivertype.JobRow{ID: 1},
		Args: ApplyRuleArgs{
			RuleID: ruleID,
		},
	}

	// Expect GetRule call
	mockStore.EXPECT().
		GetRule(gomock.Any(), ruleID).
		Return(rule, nil)

	// Expect GetView call
	mockStore.EXPECT().
		GetView(gomock.Any(), viewID).
		Return(view, nil)

	// Expect ListNotificationsFromQuery call
	mockStore.EXPECT().
		ListNotificationsFromQuery(gomock.Any(), gomock.Any()).
		Return(queryResult, nil)

	// Expect ApplyRuleActions
	mockMatcher.EXPECT().
		ApplyRuleActions(gomock.Any(), "notif-1", gomock.Any()).
		Return(nil)

	err := worker.Work(context.Background(), job)
	require.NoError(t, err)
}

func TestApplyRuleWorker_DisabledRule(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	worker, mockStore, _ := setupTestWorker(ctrl)

	ruleID := int64(1)
	rule := db.Rule{
		ID:      ruleID,
		Name:    "Test Rule",
		Enabled: false, // Disabled
		Query:   sql.NullString{String: "is:unread", Valid: true},
		ViewID:  sql.NullInt64{Valid: false},
	}

	job := &river.Job[ApplyRuleArgs]{
		JobRow: &rivertype.JobRow{ID: 1},
		Args: ApplyRuleArgs{
			RuleID: ruleID,
		},
	}

	// Expect GetRule call
	mockStore.EXPECT().
		GetRule(gomock.Any(), ruleID).
		Return(rule, nil)

	// Should return early without processing
	err := worker.Work(context.Background(), job)
	require.NoError(t, err)
}

func TestApplyRuleWorker_RuleNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	worker, mockStore, _ := setupTestWorker(ctrl)

	ruleID := int64(999)

	job := &river.Job[ApplyRuleArgs]{
		JobRow: &rivertype.JobRow{ID: 1},
		Args: ApplyRuleArgs{
			RuleID: ruleID,
		},
	}

	// Expect GetRule call to return error
	mockStore.EXPECT().
		GetRule(gomock.Any(), ruleID).
		Return(db.Rule{}, sql.ErrNoRows)

	err := worker.Work(context.Background(), job)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get rule")
}

func TestApplyRuleWorker_ViewNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	worker, mockStore, _ := setupTestWorker(ctrl)

	ruleID := int64(1)
	viewID := int64(999)
	rule := db.Rule{
		ID:      ruleID,
		Name:    "Test Rule",
		Enabled: true,
		Query:   sql.NullString{Valid: false},
		ViewID:  sql.NullInt64{Int64: viewID, Valid: true},
	}

	job := &river.Job[ApplyRuleArgs]{
		JobRow: &rivertype.JobRow{ID: 1},
		Args: ApplyRuleArgs{
			RuleID: ruleID,
		},
	}

	// Expect GetRule call
	mockStore.EXPECT().
		GetRule(gomock.Any(), ruleID).
		Return(rule, nil)

	// Expect GetView call to return error
	mockStore.EXPECT().
		GetView(gomock.Any(), viewID).
		Return(db.View{}, sql.ErrNoRows)

	err := worker.Work(context.Background(), job)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get view")
}

func TestApplyRuleWorker_ViewWithoutQuery(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	worker, mockStore, _ := setupTestWorker(ctrl)

	ruleID := int64(1)
	viewID := int64(10)
	rule := db.Rule{
		ID:      ruleID,
		Name:    "Test Rule",
		Enabled: true,
		Query:   sql.NullString{Valid: false},
		ViewID:  sql.NullInt64{Int64: viewID, Valid: true},
	}

	view := db.View{
		ID:    viewID,
		Name:  "Test View",
		Query: sql.NullString{Valid: false}, // No query
	}

	job := &river.Job[ApplyRuleArgs]{
		JobRow: &rivertype.JobRow{ID: 1},
		Args: ApplyRuleArgs{
			RuleID: ruleID,
		},
	}

	// Expect GetRule call
	mockStore.EXPECT().
		GetRule(gomock.Any(), ruleID).
		Return(rule, nil)

	// Expect GetView call
	mockStore.EXPECT().
		GetView(gomock.Any(), viewID).
		Return(view, nil)

	err := worker.Work(context.Background(), job)
	require.Error(t, err)
	require.Contains(t, err.Error(), "has no query defined")
}

func TestApplyRuleWorker_RuleWithoutQueryOrView(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	worker, mockStore, _ := setupTestWorker(ctrl)

	ruleID := int64(1)
	rule := db.Rule{
		ID:      ruleID,
		Name:    "Test Rule",
		Enabled: true,
		Query:   sql.NullString{Valid: false}, // No query
		ViewID:  sql.NullInt64{Valid: false},  // No view
	}

	job := &river.Job[ApplyRuleArgs]{
		JobRow: &rivertype.JobRow{ID: 1},
		Args: ApplyRuleArgs{
			RuleID: ruleID,
		},
	}

	// Expect GetRule call
	mockStore.EXPECT().
		GetRule(gomock.Any(), ruleID).
		Return(rule, nil)

	err := worker.Work(context.Background(), job)
	require.Error(t, err)
	require.Contains(t, err.Error(), "has neither query nor viewId set")
}

func TestApplyRuleWorker_QueryBuildError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	worker, mockStore, _ := setupTestWorker(ctrl)

	ruleID := int64(1)
	rule := db.Rule{
		ID:      ruleID,
		Name:    "Test Rule",
		Enabled: true,
		Query: sql.NullString{
			String: "badfield:value",
			Valid:  true,
		}, // Unknown field causes build error
		ViewID: sql.NullInt64{Valid: false},
	}

	job := &river.Job[ApplyRuleArgs]{
		JobRow: &rivertype.JobRow{ID: 1},
		Args: ApplyRuleArgs{
			RuleID: ruleID,
		},
	}

	// Expect GetRule call
	mockStore.EXPECT().
		GetRule(gomock.Any(), ruleID).
		Return(rule, nil)

	err := worker.Work(context.Background(), job)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to build query")
}

func TestApplyRuleWorker_ListNotificationsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	worker, mockStore, _ := setupTestWorker(ctrl)

	ruleID := int64(1)
	rule := db.Rule{
		ID:      ruleID,
		Name:    "Test Rule",
		Enabled: true,
		Query:   sql.NullString{String: "is:unread", Valid: true},
		ViewID:  sql.NullInt64{Valid: false},
	}

	job := &river.Job[ApplyRuleArgs]{
		JobRow: &rivertype.JobRow{ID: 1},
		Args: ApplyRuleArgs{
			RuleID: ruleID,
		},
	}

	// Expect GetRule call
	mockStore.EXPECT().
		GetRule(gomock.Any(), ruleID).
		Return(rule, nil)

	// Expect ListNotificationsFromQuery call to return error
	mockStore.EXPECT().
		ListNotificationsFromQuery(gomock.Any(), gomock.Any()).
		Return(db.ListNotificationsFromQueryResult{}, errors.New("database error"))

	err := worker.Work(context.Background(), job)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to execute query")
}

func TestApplyRuleWorker_EmptyNotifications(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	worker, mockStore, _ := setupTestWorker(ctrl)

	ruleID := int64(1)
	rule := db.Rule{
		ID:      ruleID,
		Name:    "Test Rule",
		Enabled: true,
		Query:   sql.NullString{String: "is:unread", Valid: true},
		ViewID:  sql.NullInt64{Valid: false},
	}

	queryResult := db.ListNotificationsFromQueryResult{
		Notifications: []db.Notification{}, // Empty
		Total:         0,
	}

	job := &river.Job[ApplyRuleArgs]{
		JobRow: &rivertype.JobRow{ID: 1},
		Args: ApplyRuleArgs{
			RuleID: ruleID,
		},
	}

	// Expect GetRule call
	mockStore.EXPECT().
		GetRule(gomock.Any(), ruleID).
		Return(rule, nil)

	// Expect ListNotificationsFromQuery call
	mockStore.EXPECT().
		ListNotificationsFromQuery(gomock.Any(), gomock.Any()).
		Return(queryResult, nil)

	// Should return early when no notifications
	err := worker.Work(context.Background(), job)
	require.NoError(t, err)
}

func TestApplyRuleWorker_ApplyRuleActionsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	worker, mockStore, mockMatcher := setupTestWorker(ctrl)

	ruleID := int64(1)
	rule := db.Rule{
		ID:      ruleID,
		Name:    "Test Rule",
		Enabled: true,
		Query:   sql.NullString{String: "is:unread", Valid: true},
		ViewID:  sql.NullInt64{Valid: false},
		Actions: json.RawMessage(`{"markRead": true}`),
	}

	notifications := []db.Notification{
		{
			ID:       1,
			GithubID: "notif-1",
		},
		{
			ID:       2,
			GithubID: "notif-2",
		},
	}

	queryResult := db.ListNotificationsFromQueryResult{
		Notifications: notifications,
		Total:         2,
	}

	job := &river.Job[ApplyRuleArgs]{
		JobRow: &rivertype.JobRow{ID: 1},
		Args: ApplyRuleArgs{
			RuleID: ruleID,
		},
	}

	// Expect GetRule call
	mockStore.EXPECT().
		GetRule(gomock.Any(), ruleID).
		Return(rule, nil)

	// Expect ListNotificationsFromQuery call (first page)
	mockStore.EXPECT().
		ListNotificationsFromQuery(gomock.Any(), gomock.Any()).
		Return(queryResult, nil)

	// First notification fails, second succeeds
	// The code processes all notifications in the result, so both will be called
	mockMatcher.EXPECT().
		ApplyRuleActions(gomock.Any(), "notif-1", gomock.Any()).
		Return(errors.New("action failed"))
	mockMatcher.EXPECT().
		ApplyRuleActions(gomock.Any(), "notif-2", gomock.Any()).
		Return(nil)

	// Expect second call for pagination (will return empty and break loop)
	emptyResult := db.ListNotificationsFromQueryResult{
		Notifications: []db.Notification{},
		Total:         2,
	}
	mockStore.EXPECT().
		ListNotificationsFromQuery(gomock.Any(), gomock.Any()).
		Return(emptyResult, nil)

	// Should continue processing even if one fails
	err := worker.Work(context.Background(), job)
	require.NoError(t, err)
}

func TestApplyRuleWorker_MultiplePages(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	worker, mockStore, mockMatcher := setupTestWorker(ctrl)

	ruleID := int64(1)
	rule := db.Rule{
		ID:      ruleID,
		Name:    "Test Rule",
		Enabled: true,
		Query:   sql.NullString{String: "is:unread", Valid: true},
		ViewID:  sql.NullInt64{Valid: false},
		Actions: json.RawMessage(`{"markRead": true}`),
	}

	// First page
	page1Notifications := []db.Notification{
		{ID: 1, GithubID: "notif-1"},
		{ID: 2, GithubID: "notif-2"},
	}
	page1Result := db.ListNotificationsFromQueryResult{
		Notifications: page1Notifications,
		Total:         3, // Total is 3, so we'll get another page
	}

	// Second page
	page2Notifications := []db.Notification{
		{ID: 3, GithubID: "notif-3"},
	}
	page2Result := db.ListNotificationsFromQueryResult{
		Notifications: page2Notifications,
		Total:         3,
	}

	job := &river.Job[ApplyRuleArgs]{
		JobRow: &rivertype.JobRow{ID: 1},
		Args: ApplyRuleArgs{
			RuleID: ruleID,
		},
	}

	// Expect GetRule call
	mockStore.EXPECT().
		GetRule(gomock.Any(), ruleID).
		Return(rule, nil)

	// Expect two ListNotificationsFromQuery calls (pagination)
	mockStore.EXPECT().
		ListNotificationsFromQuery(gomock.Any(), gomock.Any()).
		Return(page1Result, nil)
	mockStore.EXPECT().
		ListNotificationsFromQuery(gomock.Any(), gomock.Any()).
		Return(page2Result, nil)

	// Expect ApplyRuleActions for all notifications
	mockMatcher.EXPECT().
		ApplyRuleActions(gomock.Any(), "notif-1", gomock.Any()).
		Return(nil)
	mockMatcher.EXPECT().
		ApplyRuleActions(gomock.Any(), "notif-2", gomock.Any()).
		Return(nil)
	mockMatcher.EXPECT().
		ApplyRuleActions(gomock.Any(), "notif-3", gomock.Any()).
		Return(nil)

	err := worker.Work(context.Background(), job)
	require.NoError(t, err)
}

func TestApplyRuleWorker_InvalidActionsJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	worker, mockStore, _ := setupTestWorker(ctrl)

	ruleID := int64(1)
	rule := db.Rule{
		ID:      ruleID,
		Name:    "Test Rule",
		Enabled: true,
		Query:   sql.NullString{String: "is:unread", Valid: true},
		ViewID:  sql.NullInt64{Valid: false},
		Actions: json.RawMessage(`invalid json`), // Invalid JSON
	}

	notifications := []db.Notification{
		{
			ID:       1,
			GithubID: "notif-1",
		},
	}

	queryResult := db.ListNotificationsFromQueryResult{
		Notifications: notifications,
		Total:         1,
	}

	job := &river.Job[ApplyRuleArgs]{
		JobRow: &rivertype.JobRow{ID: 1},
		Args: ApplyRuleArgs{
			RuleID: ruleID,
		},
	}

	// Expect GetRule call
	mockStore.EXPECT().
		GetRule(gomock.Any(), ruleID).
		Return(rule, nil)

	// Expect ListNotificationsFromQuery call (first page)
	mockStore.EXPECT().
		ListNotificationsFromQuery(gomock.Any(), gomock.Any()).
		Return(queryResult, nil)

	// Should skip notification with invalid JSON (continue in loop)
	// No ApplyRuleActions call expected due to unmarshal error

	// Expect second call for pagination (will return empty and break loop)
	emptyResult := db.ListNotificationsFromQueryResult{
		Notifications: []db.Notification{},
		Total:         1,
	}
	mockStore.EXPECT().
		ListNotificationsFromQuery(gomock.Any(), gomock.Any()).
		Return(emptyResult, nil)

	err := worker.Work(context.Background(), job)
	require.NoError(t, err)
}

func TestApplyRuleArgs_Kind(t *testing.T) {
	args := ApplyRuleArgs{
		RuleID: 1,
	}
	require.Equal(t, "apply_rule", args.Kind())
}

func TestApplyRuleArgs_InsertOpts(t *testing.T) {
	args := ApplyRuleArgs{}
	opts := args.InsertOpts()
	require.Equal(t, "apply_rule", opts.Queue)
}

func TestNewApplyRuleWorker(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := dbmocks.NewMockStore(ctrl)
	worker := NewApplyRuleWorker(mockStore)

	require.NotNil(t, worker)
	require.Equal(t, mockStore, worker.store)
	require.NotNil(t, worker.matcher)
	// Verify it's a real RuleMatcher instance (not nil)
	_, ok := worker.matcher.(*RuleMatcher)
	require.True(t, ok)
}

func TestNewApplyRuleWorkerWithMatcher(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := dbmocks.NewMockStore(ctrl)
	mockMatcher := jobmocks.NewMockRuleMatcherInterface(ctrl)
	worker := NewApplyRuleWorkerWithMatcher(mockStore, mockMatcher)

	require.NotNil(t, worker)
	require.Equal(t, mockStore, worker.store)
	require.Equal(t, mockMatcher, worker.matcher)
}
