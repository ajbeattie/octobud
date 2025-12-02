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
	"context"
	"database/sql"

	"github.com/sqlc-dev/pqtype"
)

//go:generate mockgen -source=store.go -destination=mocks/mock_store.go -package=mocks

// Store defines the interface for database queries used by the business logic layer.
// This interface allows us to mock database operations in unit tests.
type Store interface {
	// Notification methods
	GetNotificationByGithubID(ctx context.Context, githubID string) (Notification, error)
	GetNotificationByID(ctx context.Context, id int64) (Notification, error)
	ListNotificationsFromQuery(
		ctx context.Context,
		query NotificationQuery,
	) (ListNotificationsFromQueryResult, error)
	MarkNotificationRead(ctx context.Context, githubID string) (Notification, error)
	MarkNotificationUnread(ctx context.Context, githubID string) (Notification, error)
	ArchiveNotification(ctx context.Context, githubID string) (Notification, error)
	UnarchiveNotification(ctx context.Context, githubID string) (Notification, error)
	MuteNotification(ctx context.Context, githubID string) (Notification, error)
	UnmuteNotification(ctx context.Context, githubID string) (Notification, error)
	SnoozeNotification(ctx context.Context, arg SnoozeNotificationParams) (Notification, error)
	UnsnoozeNotification(ctx context.Context, githubID string) (Notification, error)
	StarNotification(ctx context.Context, githubID string) (Notification, error)
	UnstarNotification(ctx context.Context, githubID string) (Notification, error)
	MarkNotificationFiltered(ctx context.Context, githubID string) (Notification, error)
	MarkNotificationUnfiltered(ctx context.Context, githubID string) (Notification, error)
	BulkSnoozeNotifications(ctx context.Context, arg BulkSnoozeNotificationsParams) (int64, error)
	//nolint:revive // var-naming: githubIds matches existing API contract
	BulkMarkNotificationsRead(ctx context.Context, githubIds []string) (int64, error)
	//nolint:revive // var-naming: githubIds matches existing API contract
	BulkMarkNotificationsUnread(ctx context.Context, githubIds []string) (int64, error)
	//nolint:revive // var-naming: githubIds matches existing API contract
	BulkArchiveNotifications(ctx context.Context, githubIds []string) (int64, error)
	//nolint:revive // var-naming: githubIds matches existing API contract
	BulkUnarchiveNotifications(ctx context.Context, githubIds []string) (int64, error)
	BulkMarkNotificationsReadByQuery(ctx context.Context, query NotificationQuery) (int64, error)
	BulkMarkNotificationsUnreadByQuery(ctx context.Context, query NotificationQuery) (int64, error)
	BulkArchiveNotificationsByQuery(ctx context.Context, query NotificationQuery) (int64, error)
	BulkUnarchiveNotificationsByQuery(ctx context.Context, query NotificationQuery) (int64, error)
	BulkSnoozeNotificationsByQuery(
		ctx context.Context,
		arg BulkSnoozeNotificationsByQueryParams,
	) (int64, error)
	//nolint:revive // var-naming: githubIds matches existing API contract
	BulkUnsnoozeNotifications(ctx context.Context, githubIds []string) (int64, error)
	BulkUnsnoozeNotificationsByQuery(ctx context.Context, query NotificationQuery) (int64, error)
	//nolint:revive // var-naming: githubIds matches existing API contract
	BulkMuteNotifications(ctx context.Context, githubIds []string) (int64, error)
	//nolint:revive // var-naming: githubIds matches existing API contract
	BulkUnmuteNotifications(ctx context.Context, githubIds []string) (int64, error)
	BulkMuteNotificationsByQuery(ctx context.Context, query NotificationQuery) (int64, error)
	BulkUnmuteNotificationsByQuery(ctx context.Context, query NotificationQuery) (int64, error)
	//nolint:revive // var-naming: githubIds matches existing API contract
	BulkStarNotifications(ctx context.Context, githubIds []string) (int64, error)
	//nolint:revive // var-naming: githubIds matches existing API contract
	BulkUnstarNotifications(ctx context.Context, githubIds []string) (int64, error)
	BulkStarNotificationsByQuery(ctx context.Context, query NotificationQuery) (int64, error)
	BulkUnstarNotificationsByQuery(ctx context.Context, query NotificationQuery) (int64, error)
	//nolint:revive // var-naming: githubIds matches existing API contract
	BulkMarkNotificationsUnfiltered(ctx context.Context, githubIds []string) (int64, error)
	UpdateNotificationTagIds(ctx context.Context, notificationID int64) error

	// Tag methods
	GetTag(ctx context.Context, id int64) (Tag, error)
	GetTagByName(ctx context.Context, name string) (Tag, error)
	ListAllTags(ctx context.Context) ([]Tag, error)
	UpsertTag(ctx context.Context, arg UpsertTagParams) (Tag, error)
	UpdateTag(ctx context.Context, arg UpdateTagParams) (Tag, error)
	DeleteTag(ctx context.Context, id int64) error
	UpdateTagDisplayOrder(ctx context.Context, arg UpdateTagDisplayOrderParams) error
	ListTagsForEntity(ctx context.Context, arg ListTagsForEntityParams) ([]Tag, error)
	AssignTagToEntity(ctx context.Context, arg AssignTagToEntityParams) (TagAssignment, error)
	RemoveTagAssignment(ctx context.Context, arg RemoveTagAssignmentParams) error

	// View methods
	GetView(ctx context.Context, id int64) (View, error)
	ListViews(ctx context.Context) ([]View, error)
	CreateView(ctx context.Context, arg CreateViewParams) (View, error)
	UpdateView(ctx context.Context, arg UpdateViewParams) (View, error)
	DeleteView(ctx context.Context, id int64) (int64, error)
	UpdateViewOrder(ctx context.Context, arg UpdateViewOrderParams) error
	GetRulesByViewID(ctx context.Context, viewID sql.NullInt64) ([]Rule, error)

	// Rule methods
	GetRule(ctx context.Context, id int64) (Rule, error)
	ListRules(ctx context.Context) ([]Rule, error)
	ListEnabledRulesOrdered(ctx context.Context) ([]Rule, error)
	CreateRule(ctx context.Context, arg CreateRuleParams) (Rule, error)
	UpdateRule(ctx context.Context, arg UpdateRuleParams) (Rule, error)
	DeleteRule(ctx context.Context, id int64) error
	UpdateRuleOrder(ctx context.Context, arg UpdateRuleOrderParams) error

	// Repository methods
	GetRepositoryByID(ctx context.Context, id int64) (Repository, error)
	ListRepositories(ctx context.Context) ([]Repository, error)
	UpsertRepository(ctx context.Context, arg UpsertRepositoryParams) (Repository, error)

	// Pull Request methods
	UpsertPullRequest(ctx context.Context, arg UpsertPullRequestParams) (PullRequest, error)

	// Sync methods
	GetSyncState(ctx context.Context) (GetSyncStateRow, error)
	UpsertSyncState(ctx context.Context, arg UpsertSyncStateParams) (UpsertSyncStateRow, error)

	// Notification upsert/update methods
	UpsertNotification(ctx context.Context, arg UpsertNotificationParams) (Notification, error)
	UpdateNotificationSubject(ctx context.Context, arg UpdateNotificationSubjectParams) error

	// User methods
	GetUser(ctx context.Context) (User, error)
	CreateUser(ctx context.Context, arg CreateUserParams) (User, error)
	UpdateUserUsername(ctx context.Context, username string) (User, error)
	UpdateUserPassword(ctx context.Context, passwordHash string) (User, error)
	UpdateUserCredentials(ctx context.Context, arg UpdateUserCredentialsParams) (User, error)
	UpdateUserSyncSettings(ctx context.Context, syncSettings pqtype.NullRawMessage) (User, error)
}

// Ensure Queries implements Querier interface
var _ Store = (*Queries)(nil)
