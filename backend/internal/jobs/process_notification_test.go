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

	githubmocks "github.com/ajbeattie/octobud/backend/internal/github/mocks"
	"github.com/ajbeattie/octobud/backend/internal/github/types"
)

// TestProcessNotificationWorker_Success tests successful notification processing
func TestProcessNotificationWorker_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	thread := types.NotificationThread{
		ID: "notif-123",
		Repository: types.RepositorySnapshot{
			ID:       789,
			FullName: "owner/test-repo",
			Name:     "test-repo",
		},
		Subject: types.NotificationSubject{
			Title: "Test Issue",
			Type:  "Issue",
			URL:   "https://api.github.com/repos/owner/test-repo/issues/1",
		},
		Reason:    "mention",
		Unread:    true,
		UpdatedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
	}

	threadData, err := json.Marshal(thread)
	require.NoError(t, err)

	mockSync := githubmocks.NewMockSyncOperations(ctrl)
	mockSync.EXPECT().
		ProcessNotification(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, arg types.NotificationThread) error {
			require.Equal(t, "notif-123", arg.ID)
			require.Equal(t, "owner/test-repo", arg.Repository.FullName)
			return nil
		})

	worker := NewProcessNotificationWorker(nil, mockSync)

	job := &river.Job[ProcessNotificationArgs]{
		JobRow: &rivertype.JobRow{ID: 1},
		Args: ProcessNotificationArgs{
			NotificationData: threadData,
		},
	}

	err = worker.Work(context.Background(), job)
	require.NoError(t, err)
}

// TestProcessNotificationWorker_UnmarshalError tests handling of invalid JSON
func TestProcessNotificationWorker_UnmarshalError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSync := githubmocks.NewMockSyncOperations(ctrl)
	// No ProcessNotification call expected since unmarshal fails first

	worker := NewProcessNotificationWorker(nil, mockSync)

	job := &river.Job[ProcessNotificationArgs]{
		JobRow: &rivertype.JobRow{ID: 1},
		Args: ProcessNotificationArgs{
			NotificationData: json.RawMessage(`invalid json`),
		},
	}

	err := worker.Work(context.Background(), job)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid")
}

// TestProcessNotificationWorker_ProcessError tests handling of processing errors
func TestProcessNotificationWorker_ProcessError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	thread := types.NotificationThread{
		ID: "notif-123",
		Repository: types.RepositorySnapshot{
			ID:       789,
			FullName: "owner/test-repo",
		},
		UpdatedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
	}

	threadData, err := json.Marshal(thread)
	require.NoError(t, err)

	mockSync := githubmocks.NewMockSyncOperations(ctrl)
	mockSync.EXPECT().
		ProcessNotification(gomock.Any(), gomock.Any()).
		Return(errors.New("database error"))

	worker := NewProcessNotificationWorker(nil, mockSync)

	job := &river.Job[ProcessNotificationArgs]{
		JobRow: &rivertype.JobRow{ID: 1},
		Args: ProcessNotificationArgs{
			NotificationData: threadData,
		},
	}

	err = worker.Work(context.Background(), job)
	require.Error(t, err)
	require.Contains(t, err.Error(), "database error")
}

// TestProcessNotificationWorker_EmptyData tests handling of empty notification data
func TestProcessNotificationWorker_EmptyData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSync := githubmocks.NewMockSyncOperations(ctrl)
	mockSync.EXPECT().
		ProcessNotification(gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	worker := NewProcessNotificationWorker(nil, mockSync)

	job := &river.Job[ProcessNotificationArgs]{
		JobRow: &rivertype.JobRow{ID: 1},
		Args: ProcessNotificationArgs{
			NotificationData: json.RawMessage(`{}`),
		},
	}

	err := worker.Work(context.Background(), job)

	// Empty object should unmarshal successfully but may fail processing
	// depending on validation logic
	// Current implementation will likely succeed but process an empty notification
	require.NoError(t, err)
}

// TestProcessNotificationArgs_Kind tests the Kind method
func TestProcessNotificationArgs_Kind(t *testing.T) {
	args := ProcessNotificationArgs{
		NotificationData: json.RawMessage(`{}`),
	}
	require.Equal(t, "process_notification", args.Kind())
}

// TestProcessNotificationWorker_PullRequestNotification tests processing PR notification
func TestProcessNotificationWorker_PullRequestNotification(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	thread := types.NotificationThread{
		ID: "notif-pr-123",
		Repository: types.RepositorySnapshot{
			ID:       789,
			FullName: "owner/test-repo",
			Name:     "test-repo",
		},
		Subject: types.NotificationSubject{
			Title: "Fix bug",
			Type:  "PullRequest",
			URL:   "https://api.github.com/repos/owner/test-repo/pulls/42",
		},
		Reason:    "review_requested",
		Unread:    true,
		UpdatedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
	}

	threadData, err := json.Marshal(thread)
	require.NoError(t, err)

	mockSync := githubmocks.NewMockSyncOperations(ctrl)
	mockSync.EXPECT().
		ProcessNotification(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, arg types.NotificationThread) error {
			require.Equal(t, "notif-pr-123", arg.ID)
			require.Equal(t, "PullRequest", arg.Subject.Type)
			return nil
		})

	worker := NewProcessNotificationWorker(nil, mockSync)

	job := &river.Job[ProcessNotificationArgs]{
		JobRow: &rivertype.JobRow{ID: 1},
		Args: ProcessNotificationArgs{
			NotificationData: threadData,
		},
	}

	err = worker.Work(context.Background(), job)
	require.NoError(t, err)
}

// TestProcessNotificationWorker_NilData tests handling of nil notification data
func TestProcessNotificationWorker_NilData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSync := githubmocks.NewMockSyncOperations(ctrl)
	// No ProcessNotification call expected since unmarshal fails first

	worker := NewProcessNotificationWorker(nil, mockSync)

	job := &river.Job[ProcessNotificationArgs]{
		JobRow: &rivertype.JobRow{ID: 1},
		Args: ProcessNotificationArgs{
			NotificationData: nil,
		},
	}

	err := worker.Work(context.Background(), job)

	// Nil data should result in unmarshal error
	require.Error(t, err)
}
