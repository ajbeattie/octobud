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
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/ajbeattie/octobud/backend/internal/db/mocks"
	"github.com/ajbeattie/octobud/backend/internal/github/types"
	syncmocks "github.com/ajbeattie/octobud/backend/internal/sync/mocks"
)

// TestSyncOlderNotificationsWorker_Success tests successful sync of older notifications
func TestSyncOlderNotificationsWorker_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	untilTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	sinceTime := untilTime.AddDate(0, 0, -30) // 30 days back

	notifications := []types.NotificationThread{
		{
			ID: "notif-old-1",
			Repository: types.RepositorySnapshot{
				ID:       123,
				FullName: "test/repo1",
			},
			UpdatedAt: time.Date(2024, 1, 10, 10, 0, 0, 0, time.UTC),
		},
		{
			ID: "notif-old-2",
			Repository: types.RepositorySnapshot{
				ID:       456,
				FullName: "test/repo2",
			},
			UpdatedAt: time.Date(2024, 1, 5, 10, 0, 0, 0, time.UTC),
		},
	}

	mockSync := syncmocks.NewMockSyncOperations(ctrl)
	mockSync.EXPECT().
		FetchOlderNotificationsToSync(gomock.Any(), sinceTime, untilTime, (*int)(nil), false).
		Return(notifications, nil)
	mockSync.EXPECT().
		UpdateSyncStateAfterProcessingWithInitialSync(
			gomock.Any(),
			time.Time{}, // Don't update latest_notification_at
			(*time.Time)(nil),
			gomock.Any(), // oldest notification
		).
		Return(nil)

	mockRiver := mocks.NewMockRiverClient(ctrl)
	// Expect two Insert calls, one for each notification
	mockRiver.EXPECT().
		Insert(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, args river.JobArgs, _ *river.InsertOpts) (*rivertype.JobInsertResult, error) {
			processArgs, ok := args.(ProcessNotificationArgs)
			require.True(t, ok)
			require.NotNil(t, processArgs.NotificationData)

			var thread types.NotificationThread
			err := json.Unmarshal(processArgs.NotificationData, &thread)
			require.NoError(t, err)
			require.Contains(t, []string{"notif-old-1", "notif-old-2"}, thread.ID)

			return &rivertype.JobInsertResult{Job: &rivertype.JobRow{ID: 1}}, nil
		}).
		Times(2)

	worker := NewSyncOlderNotificationsWorker(zap.NewNop(), mockSync, mockRiver)

	job := &river.Job[SyncOlderNotificationsArgs]{
		JobRow: &rivertype.JobRow{ID: 1},
		Args: SyncOlderNotificationsArgs{
			Days:       30,
			UntilTime:  untilTime,
			MaxCount:   nil,
			UnreadOnly: false,
		},
	}

	err := worker.Work(context.Background(), job)
	require.NoError(t, err)
}

// TestSyncOlderNotificationsWorker_EmptyResults tests when no older notifications are found
func TestSyncOlderNotificationsWorker_EmptyResults(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	untilTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	sinceTime := untilTime.AddDate(0, 0, -30)

	mockSync := syncmocks.NewMockSyncOperations(ctrl)
	mockSync.EXPECT().
		FetchOlderNotificationsToSync(gomock.Any(), sinceTime, untilTime, (*int)(nil), false).
		Return([]types.NotificationThread{}, nil)
	// No UpdateSyncStateAfterProcessingWithInitialSync call expected for empty results

	mockRiver := mocks.NewMockRiverClient(ctrl)
	// No Insert calls expected for empty results

	worker := NewSyncOlderNotificationsWorker(zap.NewNop(), mockSync, mockRiver)

	job := &river.Job[SyncOlderNotificationsArgs]{
		JobRow: &rivertype.JobRow{ID: 1},
		Args: SyncOlderNotificationsArgs{
			Days:       30,
			UntilTime:  untilTime,
			MaxCount:   nil,
			UnreadOnly: false,
		},
	}

	err := worker.Work(context.Background(), job)
	require.NoError(t, err)
}

// TestSyncOlderNotificationsWorker_WithMaxCount tests max count limit is passed correctly
func TestSyncOlderNotificationsWorker_WithMaxCount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	untilTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	sinceTime := untilTime.AddDate(0, 0, -30)
	maxCount := 100

	notifications := []types.NotificationThread{
		{
			ID:        "notif-old-1",
			UpdatedAt: time.Date(2024, 1, 10, 10, 0, 0, 0, time.UTC),
		},
	}

	mockSync := syncmocks.NewMockSyncOperations(ctrl)
	mockSync.EXPECT().
		FetchOlderNotificationsToSync(gomock.Any(), sinceTime, untilTime, &maxCount, false).
		Return(notifications, nil)
	mockSync.EXPECT().
		UpdateSyncStateAfterProcessingWithInitialSync(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)

	mockRiver := mocks.NewMockRiverClient(ctrl)
	mockRiver.EXPECT().
		Insert(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&rivertype.JobInsertResult{Job: &rivertype.JobRow{ID: 1}}, nil)

	worker := NewSyncOlderNotificationsWorker(zap.NewNop(), mockSync, mockRiver)

	job := &river.Job[SyncOlderNotificationsArgs]{
		JobRow: &rivertype.JobRow{ID: 1},
		Args: SyncOlderNotificationsArgs{
			Days:       30,
			UntilTime:  untilTime,
			MaxCount:   &maxCount,
			UnreadOnly: false,
		},
	}

	err := worker.Work(context.Background(), job)
	require.NoError(t, err)
}

// TestSyncOlderNotificationsWorker_WithUnreadOnly tests unread only flag is passed correctly
func TestSyncOlderNotificationsWorker_WithUnreadOnly(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	untilTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	sinceTime := untilTime.AddDate(0, 0, -30)

	notifications := []types.NotificationThread{
		{
			ID:        "notif-old-1",
			UpdatedAt: time.Date(2024, 1, 10, 10, 0, 0, 0, time.UTC),
			Unread:    true,
		},
	}

	mockSync := syncmocks.NewMockSyncOperations(ctrl)
	mockSync.EXPECT().
		FetchOlderNotificationsToSync(gomock.Any(), sinceTime, untilTime, (*int)(nil), true).
		// unreadOnly=true
		Return(notifications, nil)
	mockSync.EXPECT().
		UpdateSyncStateAfterProcessingWithInitialSync(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)

	mockRiver := mocks.NewMockRiverClient(ctrl)
	mockRiver.EXPECT().
		Insert(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&rivertype.JobInsertResult{Job: &rivertype.JobRow{ID: 1}}, nil)

	worker := NewSyncOlderNotificationsWorker(zap.NewNop(), mockSync, mockRiver)

	job := &river.Job[SyncOlderNotificationsArgs]{
		JobRow: &rivertype.JobRow{ID: 1},
		Args: SyncOlderNotificationsArgs{
			Days:       30,
			UntilTime:  untilTime,
			MaxCount:   nil,
			UnreadOnly: true,
		},
	}

	err := worker.Work(context.Background(), job)
	require.NoError(t, err)
}

// TestSyncOlderNotificationsWorker_FetchError tests error handling when fetching fails
func TestSyncOlderNotificationsWorker_FetchError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	untilTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	sinceTime := untilTime.AddDate(0, 0, -30)

	mockSync := syncmocks.NewMockSyncOperations(ctrl)
	mockSync.EXPECT().
		FetchOlderNotificationsToSync(gomock.Any(), sinceTime, untilTime, (*int)(nil), false).
		Return(nil, errors.New("API error"))

	mockRiver := mocks.NewMockRiverClient(ctrl)
	// No Insert calls expected when fetch fails

	worker := NewSyncOlderNotificationsWorker(zap.NewNop(), mockSync, mockRiver)

	job := &river.Job[SyncOlderNotificationsArgs]{
		JobRow: &rivertype.JobRow{ID: 1},
		Args: SyncOlderNotificationsArgs{
			Days:       30,
			UntilTime:  untilTime,
			MaxCount:   nil,
			UnreadOnly: false,
		},
	}

	err := worker.Work(context.Background(), job)
	require.Error(t, err)
	require.Contains(t, err.Error(), "API error")
}

// TestSyncOlderNotificationsWorker_QueueingFailure tests partial success when queuing fails
func TestSyncOlderNotificationsWorker_QueueingFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	untilTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	sinceTime := untilTime.AddDate(0, 0, -30)

	notifications := []types.NotificationThread{
		{
			ID:        "notif-old-1",
			UpdatedAt: time.Date(2024, 1, 10, 10, 0, 0, 0, time.UTC),
		},
	}

	mockSync := syncmocks.NewMockSyncOperations(ctrl)
	mockSync.EXPECT().
		FetchOlderNotificationsToSync(gomock.Any(), sinceTime, untilTime, (*int)(nil), false).
		Return(notifications, nil)
	// No sync state update expected when queueing fails (no successful inserts)

	mockRiver := mocks.NewMockRiverClient(ctrl)
	mockRiver.EXPECT().
		Insert(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, errors.New("queue full"))

	worker := NewSyncOlderNotificationsWorker(zap.NewNop(), mockSync, mockRiver)

	job := &river.Job[SyncOlderNotificationsArgs]{
		JobRow: &rivertype.JobRow{ID: 1},
		Args: SyncOlderNotificationsArgs{
			Days:       30,
			UntilTime:  untilTime,
			MaxCount:   nil,
			UnreadOnly: false,
		},
	}

	// Should not error even though queue insertion failed
	// (based on current implementation that continues on error)
	err := worker.Work(context.Background(), job)
	require.NoError(t, err)
}

// TestSyncOlderNotificationsWorker_SyncStateUpdateError tests when sync state update fails
func TestSyncOlderNotificationsWorker_SyncStateUpdateError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	untilTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	sinceTime := untilTime.AddDate(0, 0, -30)

	notifications := []types.NotificationThread{
		{
			ID:        "notif-old-1",
			UpdatedAt: time.Date(2024, 1, 10, 10, 0, 0, 0, time.UTC),
		},
	}

	mockSync := syncmocks.NewMockSyncOperations(ctrl)
	mockSync.EXPECT().
		FetchOlderNotificationsToSync(gomock.Any(), sinceTime, untilTime, (*int)(nil), false).
		Return(notifications, nil)
	mockSync.EXPECT().
		UpdateSyncStateAfterProcessingWithInitialSync(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(errors.New("database error"))

	mockRiver := mocks.NewMockRiverClient(ctrl)
	mockRiver.EXPECT().
		Insert(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&rivertype.JobInsertResult{Job: &rivertype.JobRow{ID: 1}}, nil)

	worker := NewSyncOlderNotificationsWorker(zap.NewNop(), mockSync, mockRiver)

	job := &river.Job[SyncOlderNotificationsArgs]{
		JobRow: &rivertype.JobRow{ID: 1},
		Args: SyncOlderNotificationsArgs{
			Days:       30,
			UntilTime:  untilTime,
			MaxCount:   nil,
			UnreadOnly: false,
		},
	}

	// Should not error even when sync state update fails
	// Jobs are already queued, so we don't want to fail the entire job
	err := worker.Work(context.Background(), job)
	require.NoError(t, err)
}

// TestSyncOlderNotificationsWorker_TracksOldestNotification tests that oldest notification is tracked correctly
func TestSyncOlderNotificationsWorker_TracksOldestNotification(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	untilTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	sinceTime := untilTime.AddDate(0, 0, -30)

	// Oldest notification should be Jan 5
	oldestTime := time.Date(2024, 1, 5, 10, 0, 0, 0, time.UTC)
	notifications := []types.NotificationThread{
		{
			ID:        "notif-old-1",
			UpdatedAt: time.Date(2024, 1, 10, 10, 0, 0, 0, time.UTC),
		},
		{
			ID:        "notif-old-2",
			UpdatedAt: oldestTime, // Oldest
		},
		{
			ID:        "notif-old-3",
			UpdatedAt: time.Date(2024, 1, 8, 10, 0, 0, 0, time.UTC),
		},
	}

	mockSync := syncmocks.NewMockSyncOperations(ctrl)
	mockSync.EXPECT().
		FetchOlderNotificationsToSync(gomock.Any(), sinceTime, untilTime, (*int)(nil), false).
		Return(notifications, nil)
	mockSync.EXPECT().
		UpdateSyncStateAfterProcessingWithInitialSync(
			gomock.Any(),
			time.Time{},
			(*time.Time)(nil),
			gomock.Any(),
		).
		DoAndReturn(func(_ context.Context, _ time.Time, _ *time.Time, oldest *time.Time) error {
			require.NotNil(t, oldest)
			require.Equal(t, oldestTime, *oldest)
			return nil
		})

	mockRiver := mocks.NewMockRiverClient(ctrl)
	mockRiver.EXPECT().
		Insert(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&rivertype.JobInsertResult{Job: &rivertype.JobRow{ID: 1}}, nil).
		Times(3)

	worker := NewSyncOlderNotificationsWorker(zap.NewNop(), mockSync, mockRiver)

	job := &river.Job[SyncOlderNotificationsArgs]{
		JobRow: &rivertype.JobRow{ID: 1},
		Args: SyncOlderNotificationsArgs{
			Days:       30,
			UntilTime:  untilTime,
			MaxCount:   nil,
			UnreadOnly: false,
		},
	}

	err := worker.Work(context.Background(), job)
	require.NoError(t, err)
}

// TestSyncOlderNotificationsArgs_Kind tests the Kind method
func TestSyncOlderNotificationsArgs_Kind(t *testing.T) {
	args := SyncOlderNotificationsArgs{}
	require.Equal(t, "sync_older_notifications", args.Kind())
}

// TestSyncOlderNotificationsArgs_InsertOpts tests the InsertOpts method
func TestSyncOlderNotificationsArgs_InsertOpts(t *testing.T) {
	args := SyncOlderNotificationsArgs{}
	opts := args.InsertOpts()
	require.Equal(t, "sync_notifications", opts.Queue)
}
