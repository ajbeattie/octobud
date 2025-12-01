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

package github

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ajbeattie/octobud/backend/internal/core/pullrequest"
	"github.com/ajbeattie/octobud/backend/internal/core/repository"
	"github.com/ajbeattie/octobud/backend/internal/core/sync"
	"github.com/ajbeattie/octobud/backend/internal/db"
	githubinterfaces "github.com/ajbeattie/octobud/backend/internal/github/interfaces"
	githubmocks "github.com/ajbeattie/octobud/backend/internal/github/mocks"
	"github.com/ajbeattie/octobud/backend/internal/github/types"
)

// mockClock returns a fixed time for testing
func mockClock() time.Time {
	return time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
}

// setupSyncService creates a SyncService with mocked dependencies for testing
//

func setupSyncService(
	_ *testing.T,
	dbConn *sql.DB,
	mockClient githubinterfaces.Client,
	opts ...SyncOption,
) *SyncService {
	queries := db.New(dbConn)
	syncService := sync.NewService(queries)
	repositoryService := repository.NewService(queries)
	pullRequestService := pullrequest.NewService(queries)

	allOpts := append([]SyncOption{WithClock(mockClock)}, opts...)
	return NewSyncService(
		mockClient,
		syncService,
		repositoryService,
		pullRequestService,
		queries,
		allOpts...)
}

// TestFetchNotificationsToSync_InitialSync tests fetching notifications when no sync state exists
func TestFetchNotificationsToSync_InitialSync(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dbConn, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer dbConn.Close()

	notifications := []types.NotificationThread{
		{
			ID: "notif-1",
			Repository: types.RepositorySnapshot{
				ID:       123,
				FullName: "test/repo",
			},
			Subject: types.NotificationSubject{
				Title: "Test PR",
				Type:  "PullRequest",
			},
			UpdatedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		},
	}

	mockClient := githubmocks.NewMockClient(ctrl)
	mockClient.EXPECT().
		FetchNotifications(gomock.Any(), gomock.Any()).
		Return(notifications, nil)

	// Expect GetSyncState to return no rows (initial sync)
	mock.ExpectQuery(`SELECT (.+) FROM sync_state`).
		WillReturnError(sql.ErrNoRows)

	service := setupSyncService(t, dbConn, mockClient)

	threads, err := service.FetchNotificationsToSync(context.Background())

	require.NoError(t, err)
	require.Len(t, threads, 1)
	require.Equal(t, "notif-1", threads[0].ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestFetchNotificationsToSync_WithExistingState tests fetching with existing sync state
func TestFetchNotificationsToSync_WithExistingState(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dbConn, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer dbConn.Close()

	lastSync := time.Date(2024, 1, 14, 10, 0, 0, 0, time.UTC)
	notifications := []types.NotificationThread{
		{
			ID: "notif-2",
			Repository: types.RepositorySnapshot{
				ID:       456,
				FullName: "test/repo2",
			},
			UpdatedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		},
	}

	var capturedSince *time.Time
	mockClient := githubmocks.NewMockClient(ctrl)
	mockClient.EXPECT().
		FetchNotifications(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, since *time.Time) ([]types.NotificationThread, error) {
			capturedSince = since
			return notifications, nil
		})

	// Return existing sync state (all 6 fields)
	rows := sqlmock.NewRows([]string{
		"id", "last_successful_poll", "latest_notification_at",
		"last_notification_etag", "created_at", "updated_at",
	}).AddRow(
		1,
		sql.NullTime{Time: lastSync, Valid: true},
		sql.NullTime{},
		sql.NullString{},
		time.Now(),
		time.Now(),
	)

	mock.ExpectQuery(`SELECT (.+) FROM sync_state`).
		WillReturnRows(rows)

	service := setupSyncService(t, dbConn, mockClient)

	threads, err := service.FetchNotificationsToSync(context.Background())

	require.NoError(t, err)
	require.Len(t, threads, 1)
	require.Equal(t, "notif-2", threads[0].ID)
	require.NotNil(t, capturedSince)
	require.Equal(t, lastSync, *capturedSince)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestFetchNotificationsToSync_EmptyResults tests when GitHub returns no new notifications
func TestFetchNotificationsToSync_EmptyResults(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dbConn, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer dbConn.Close()

	mockClient := githubmocks.NewMockClient(ctrl)
	mockClient.EXPECT().
		FetchNotifications(gomock.Any(), gomock.Any()).
		Return([]types.NotificationThread{}, nil)

	mock.ExpectQuery(`SELECT (.+) FROM sync_state`).
		WillReturnError(sql.ErrNoRows)

	// Expect UpsertSyncState to be called even with no notifications (all 6 return fields)
	mock.ExpectQuery(`INSERT INTO sync_state`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "last_successful_poll", "latest_notification_at",
			"last_notification_etag", "created_at", "updated_at",
		}).AddRow(1, time.Now(), sql.NullTime{}, sql.NullString{}, time.Now(), time.Now()))

	service := setupSyncService(t, dbConn, mockClient)

	threads, err := service.FetchNotificationsToSync(context.Background())

	require.NoError(t, err)
	require.Len(t, threads, 0)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestFetchNotificationsToSync_ClientError tests handling of GitHub client errors
func TestFetchNotificationsToSync_ClientError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dbConn, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer dbConn.Close()

	mockClient := githubmocks.NewMockClient(ctrl)
	mockClient.EXPECT().
		FetchNotifications(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("API error"))

	mock.ExpectQuery(`SELECT (.+) FROM sync_state`).
		WillReturnError(sql.ErrNoRows)

	service := setupSyncService(t, dbConn, mockClient)

	threads, err := service.FetchNotificationsToSync(context.Background())

	require.Error(t, err)
	require.Contains(t, err.Error(), "fetch notifications")
	require.Nil(t, threads)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestUpdateSyncStateAfterProcessing_Success tests successful sync state update
func TestUpdateSyncStateAfterProcessing_Success(t *testing.T) {
	dbConn, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer dbConn.Close()

	latestUpdate := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	mock.ExpectQuery(`INSERT INTO sync_state`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "last_successful_poll", "latest_notification_at",
			"last_notification_etag", "created_at", "updated_at",
		}).AddRow(1, time.Now(), latestUpdate, sql.NullString{}, time.Now(), time.Now()))

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := githubmocks.NewMockClient(ctrl)
	service := setupSyncService(t, dbConn, mockClient)

	err = service.UpdateSyncStateAfterProcessing(context.Background(), latestUpdate)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestUpdateSyncStateAfterProcessing_ZeroTime tests update with zero time
func TestUpdateSyncStateAfterProcessing_ZeroTime(t *testing.T) {
	dbConn, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer dbConn.Close()

	mock.ExpectQuery(`INSERT INTO sync_state`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "last_successful_poll", "latest_notification_at",
			"last_notification_etag", "created_at", "updated_at",
		}).AddRow(1, time.Now(), sql.NullTime{}, sql.NullString{}, time.Now(), time.Now()))

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := githubmocks.NewMockClient(ctrl)
	service := setupSyncService(t, dbConn, mockClient)

	err = service.UpdateSyncStateAfterProcessing(context.Background(), time.Time{})

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestProcessNotification_Success tests successful notification processing
func TestProcessNotification_Success(t *testing.T) {
	// This test would require extensive mocking of sqlc-generated queries
	// Instead, we rely on integration tests for full ProcessNotification coverage
	// Unit tests focus on helper functions and smaller units of work
	t.Skip("ProcessNotification requires integration testing due to complex sqlc mocking")
}

// TestProcessNotification_WithPullRequest tests processing PR notification
func TestProcessNotification_WithPullRequest(t *testing.T) {
	// This test would require extensive mocking of sqlc-generated queries
	// Instead, we rely on integration tests for full ProcessNotification coverage
	// Unit tests focus on helper functions and smaller units of work
	t.Skip("ProcessNotification with PR requires integration testing due to complex sqlc mocking")
}

// TestProcessNotification_RepositoryError tests repository upsert failure
func TestProcessNotification_RepositoryError(t *testing.T) {
	dbConn, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer dbConn.Close()

	thread := types.NotificationThread{
		ID: "notif-123",
		Repository: types.RepositorySnapshot{
			ID:       789,
			FullName: "owner/test-repo",
			Name:     "test-repo",
		},
		Subject: types.NotificationSubject{
			Title: "Test",
			Type:  "Issue",
		},
		UpdatedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := githubmocks.NewMockClient(ctrl)

	// Repository upsert fails
	mock.ExpectQuery(`INSERT INTO repositories`).
		WillReturnError(errors.New("database error"))

	service := setupSyncService(t, dbConn, mockClient)

	err = service.ProcessNotification(context.Background(), thread)

	require.Error(t, err)
	require.Contains(t, err.Error(), "upsert repository")
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestRefreshSubjectData_Success tests successful subject data refresh
func TestRefreshSubjectData_Success(t *testing.T) {
	// This test would require extensive mocking of sqlc-generated queries
	// Instead, we rely on integration tests for full RefreshSubjectData coverage
	// Unit tests focus on helper functions and smaller units of work
	t.Skip("RefreshSubjectData requires integration testing due to complex sqlc mocking")
}

// TestRefreshSubjectData_NotificationNotFound tests error when notification doesn't exist
func TestRefreshSubjectData_NotificationNotFound(t *testing.T) {
	dbConn, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer dbConn.Close()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := githubmocks.NewMockClient(ctrl)

	mock.ExpectQuery(`SELECT (.+) FROM notifications WHERE github_id`).
		WithArgs("nonexistent").
		WillReturnError(sql.ErrNoRows)

	service := setupSyncService(t, dbConn, mockClient)

	err = service.RefreshSubjectData(context.Background(), "nonexistent")

	require.Error(t, err)
	require.Contains(t, err.Error(), "get notification")
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestExtractAuthorFromSubject_UserField tests author extraction from user field
func TestExtractAuthorFromSubject_UserField(t *testing.T) {
	subjectJSON := json.RawMessage(`{
		"user": {
			"login": "octocat",
			"id": 123
		}
	}`)

	login, id := ExtractAuthorFromSubject(subjectJSON)

	require.True(t, login.Valid)
	require.Equal(t, "octocat", login.String)
	require.True(t, id.Valid)
	require.Equal(t, int64(123), id.Int64)
}

// TestExtractAuthorFromSubject_SenderField tests author extraction from sender field
func TestExtractAuthorFromSubject_SenderField(t *testing.T) {
	subjectJSON := json.RawMessage(`{
		"sender": {
			"login": "bot",
			"id": 456
		}
	}`)

	login, id := ExtractAuthorFromSubject(subjectJSON)

	require.True(t, login.Valid)
	require.Equal(t, "bot", login.String)
	require.True(t, id.Valid)
	require.Equal(t, int64(456), id.Int64)
}

// TestExtractAuthorFromSubject_NoAuthor tests when no author field is present
func TestExtractAuthorFromSubject_NoAuthor(t *testing.T) {
	subjectJSON := json.RawMessage(`{"title": "No author"}`)

	login, id := ExtractAuthorFromSubject(subjectJSON)

	require.False(t, login.Valid)
	require.False(t, id.Valid)
}

// TestExtractAuthorFromSubject_InvalidJSON tests handling of invalid JSON
func TestExtractAuthorFromSubject_InvalidJSON(t *testing.T) {
	subjectJSON := json.RawMessage(`invalid json`)

	login, id := ExtractAuthorFromSubject(subjectJSON)

	require.False(t, login.Valid)
	require.False(t, id.Valid)
}
