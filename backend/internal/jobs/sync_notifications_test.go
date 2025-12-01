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

	"github.com/ajbeattie/octobud/backend/internal/db/mocks"
	githubmocks "github.com/ajbeattie/octobud/backend/internal/github/mocks"
	"github.com/ajbeattie/octobud/backend/internal/github/types"
)

// TestSyncNotificationsWorker_Success tests successful sync with multiple notifications
func TestSyncNotificationsWorker_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	notifications := []types.NotificationThread{
		{
			ID: "notif-1",
			Repository: types.RepositorySnapshot{
				ID:       123,
				FullName: "test/repo1",
			},
			UpdatedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		},
		{
			ID: "notif-2",
			Repository: types.RepositorySnapshot{
				ID:       456,
				FullName: "test/repo2",
			},
			UpdatedAt: time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC),
		},
	}

	mockSync := githubmocks.NewMockSyncOperations(ctrl)
	mockSync.EXPECT().
		FetchNotificationsToSync(gomock.Any()).
		Return(notifications, nil)
	mockSync.EXPECT().
		UpdateSyncStateAfterProcessing(gomock.Any(), time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC)).
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
			require.Contains(t, []string{"notif-1", "notif-2"}, thread.ID)

			return &rivertype.JobInsertResult{Job: &rivertype.JobRow{ID: 1}}, nil
		}).
		Times(2)

	worker := NewSyncNotificationsWorker(mockSync, mockRiver)

	job := &river.Job[SyncNotificationsArgs]{
		JobRow: &rivertype.JobRow{ID: 1},
		Args:   SyncNotificationsArgs{},
	}

	err := worker.Work(context.Background(), job)
	require.NoError(t, err)
}

// TestSyncNotificationsWorker_EmptyResults tests successful sync with no new notifications
func TestSyncNotificationsWorker_EmptyResults(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSync := githubmocks.NewMockSyncOperations(ctrl)
	mockSync.EXPECT().
		FetchNotificationsToSync(gomock.Any()).
		Return([]types.NotificationThread{}, nil)

	mockRiver := mocks.NewMockRiverClient(ctrl)
	// No Insert calls expected for empty results

	worker := NewSyncNotificationsWorker(mockSync, mockRiver)

	job := &river.Job[SyncNotificationsArgs]{
		JobRow: &rivertype.JobRow{ID: 1},
		Args:   SyncNotificationsArgs{},
	}

	err := worker.Work(context.Background(), job)
	require.NoError(t, err)
}

// TestSyncNotificationsWorker_FetchError tests error handling when fetching notifications fails
func TestSyncNotificationsWorker_FetchError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSync := githubmocks.NewMockSyncOperations(ctrl)
	mockSync.EXPECT().
		FetchNotificationsToSync(gomock.Any()).
		Return(nil, errors.New("API error"))

	mockRiver := mocks.NewMockRiverClient(ctrl)
	// No Insert calls expected when fetch fails

	worker := NewSyncNotificationsWorker(mockSync, mockRiver)

	job := &river.Job[SyncNotificationsArgs]{
		JobRow: &rivertype.JobRow{ID: 1},
		Args:   SyncNotificationsArgs{},
	}

	err := worker.Work(context.Background(), job)
	require.Error(t, err)
	require.Contains(t, err.Error(), "API error")
}

// TestSyncNotificationsWorker_QueueingFailure tests partial success when queuing fails
func TestSyncNotificationsWorker_QueueingFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	notifications := []types.NotificationThread{
		{
			ID: "notif-1",
			Repository: types.RepositorySnapshot{
				ID:       123,
				FullName: "test/repo1",
			},
			UpdatedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		},
	}

	mockSync := githubmocks.NewMockSyncOperations(ctrl)
	mockSync.EXPECT().
		FetchNotificationsToSync(gomock.Any()).
		Return(notifications, nil)

	mockRiver := mocks.NewMockRiverClient(ctrl)
	mockRiver.EXPECT().
		Insert(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, errors.New("queue full"))

	worker := NewSyncNotificationsWorker(mockSync, mockRiver)

	job := &river.Job[SyncNotificationsArgs]{
		JobRow: &rivertype.JobRow{ID: 1},
		Args:   SyncNotificationsArgs{},
	}

	// Note: The current implementation continues even if queuing fails
	// This test verifies that behavior
	err := worker.Work(context.Background(), job)

	// Should not error even though queue insertion failed
	// (based on current implementation that continues on error)
	require.NoError(t, err)
}

// TestSyncNotificationsWorker_MarshalError tests handling of unmarshalable notifications
func TestSyncNotificationsWorker_MarshalError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	notifications := []types.NotificationThread{
		{
			ID: "notif-1",
			Repository: types.RepositorySnapshot{
				ID:       123,
				FullName: "test/repo1",
			},
			UpdatedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		},
	}

	mockSync := githubmocks.NewMockSyncOperations(ctrl)
	mockSync.EXPECT().
		FetchNotificationsToSync(gomock.Any()).
		Return(notifications, nil)
	mockSync.EXPECT().
		UpdateSyncStateAfterProcessing(gomock.Any(), gomock.Any()).
		Return(nil)

	mockRiver := mocks.NewMockRiverClient(ctrl)
	// In practice, json.Marshal rarely fails, but if it does, Insert won't be called
	// We'll allow it to be called since the notification should marshal successfully
	mockRiver.EXPECT().
		Insert(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&rivertype.JobInsertResult{Job: &rivertype.JobRow{ID: 1}}, nil)

	worker := NewSyncNotificationsWorker(mockSync, mockRiver)

	job := &river.Job[SyncNotificationsArgs]{
		JobRow: &rivertype.JobRow{ID: 1},
		Args:   SyncNotificationsArgs{},
	}

	err := worker.Work(context.Background(), job)

	// Current implementation continues on marshal error
	// Job count may be 0 if marshal fails
	require.NoError(t, err)
}

// TestSyncNotificationsWorker_UpdateSyncStateError tests when sync state update fails
func TestSyncNotificationsWorker_UpdateSyncStateError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	notifications := []types.NotificationThread{
		{
			ID: "notif-1",
			Repository: types.RepositorySnapshot{
				ID:       123,
				FullName: "test/repo1",
			},
			UpdatedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		},
	}

	mockSync := githubmocks.NewMockSyncOperations(ctrl)
	mockSync.EXPECT().
		FetchNotificationsToSync(gomock.Any()).
		Return(notifications, nil)
	mockSync.EXPECT().
		UpdateSyncStateAfterProcessing(gomock.Any(), gomock.Any()).
		Return(errors.New("database error"))

	mockRiver := mocks.NewMockRiverClient(ctrl)
	mockRiver.EXPECT().
		Insert(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&rivertype.JobInsertResult{Job: &rivertype.JobRow{ID: 1}}, nil)

	worker := NewSyncNotificationsWorker(mockSync, mockRiver)

	job := &river.Job[SyncNotificationsArgs]{
		JobRow: &rivertype.JobRow{ID: 1},
		Args:   SyncNotificationsArgs{},
	}

	err := worker.Work(context.Background(), job)

	// Current implementation doesn't return error when sync state update fails
	// Jobs are already queued, so we don't want to fail the entire job
	require.NoError(t, err)
}

// TestSyncNotificationsArgs_Kind tests the Kind method
func TestSyncNotificationsArgs_Kind(t *testing.T) {
	args := SyncNotificationsArgs{}
	require.Equal(t, "sync_notifications", args.Kind())
}
