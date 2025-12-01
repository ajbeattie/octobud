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

package notification

import (
	"context"

	"github.com/ajbeattie/octobud/backend/internal/db"
	"github.com/ajbeattie/octobud/backend/internal/models"
	"github.com/ajbeattie/octobud/backend/internal/query/eval"
)

// NotificationReader defines read operations for notifications
//
//nolint:revive // exported type name stutters with package name
type NotificationReader interface {
	ListNotifications(
		ctx context.Context,
		opts models.ListOptions,
	) (models.ListDetailsResult, error)
	ListPollNotifications(
		ctx context.Context,
		opts models.ListOptions,
	) (models.ListPollResult, error)
	GetByGithubID(ctx context.Context, githubID string) (db.Notification, error)
	ListNotificationsFromQueryString(
		ctx context.Context,
		queryStr string,
		limit int32,
	) ([]db.Notification, error)
	GetTagsForNotification(ctx context.Context, notificationID int64) ([]db.Tag, error)
	NewEvaluator(queryStr string) (*eval.Evaluator, error)
	GetNotificationWithDetails(
		ctx context.Context,
		githubID string,
		queryStr string,
	) (models.Notification, error)
	BuildResponse(
		ctx context.Context,
		notification db.Notification,
		repoMap map[int64]db.Repository,
		evaluator *eval.Evaluator,
	) (models.Notification, error)
	IndexRepositories(ctx context.Context) (map[int64]db.Repository, error)
}

// NotificationWriter defines individual write operations for notifications
//
//nolint:revive // exported type name stutters with package name
type NotificationWriter interface {
	UpsertNotification(
		ctx context.Context,
		params db.UpsertNotificationParams,
	) (db.Notification, error)
	UpdateNotificationSubject(ctx context.Context, params db.UpdateNotificationSubjectParams) error
	MarkNotificationRead(ctx context.Context, githubID string) (db.Notification, error)
	MarkNotificationUnread(ctx context.Context, githubID string) (db.Notification, error)
	ArchiveNotification(ctx context.Context, githubID string) (db.Notification, error)
	UnarchiveNotification(ctx context.Context, githubID string) (db.Notification, error)
	SnoozeNotification(
		ctx context.Context,
		githubID string,
		snoozedUntil string,
	) (db.Notification, error)
	UnsnoozeNotification(ctx context.Context, githubID string) (db.Notification, error)
	MuteNotification(ctx context.Context, githubID string) (db.Notification, error)
	UnmuteNotification(ctx context.Context, githubID string) (db.Notification, error)
	StarNotification(ctx context.Context, githubID string) (db.Notification, error)
	UnstarNotification(ctx context.Context, githubID string) (db.Notification, error)
	UnfilterNotification(ctx context.Context, githubID string) (db.Notification, error)
}

// NotificationTagger defines tag operations for notifications
//
//nolint:revive // exported type name stutters with package name
type NotificationTagger interface {
	AssignTag(ctx context.Context, githubID string, tagID int64) (db.Notification, error)
	AssignTagByName(ctx context.Context, githubID string, tagName string) (db.Notification, error)
	RemoveTag(ctx context.Context, githubID string, tagID int64) (db.Notification, error)
}

// BulkOperations defines bulk operations for notifications
type BulkOperations interface {
	BulkAssignTag(ctx context.Context, notifications []db.Notification, tagID int64) (int, error)
	BulkRemoveTag(ctx context.Context, notifications []db.Notification, tagID int64) (int, error)
	BulkUpdate(
		ctx context.Context,
		op models.BulkOperationType,
		target models.BulkOperationTarget,
		params models.BulkUpdateParams,
	) (int64, error)
}

// NotificationService is the composed interface containing all notification operations.
// It combines Reader, Writer, Tagger, and BulkOperations interfaces.
//
//nolint:revive // exported type name stutters with package name
type NotificationService interface {
	NotificationReader
	NotificationWriter
	NotificationTagger
	BulkOperations
}

// Service provides higher-level operations over notification records.
type Service struct {
	queries db.Store
}

// NewService constructs a Service backed by the provided queries.
func NewService(queries db.Store) *Service {
	return &Service{
		queries: queries,
	}
}
