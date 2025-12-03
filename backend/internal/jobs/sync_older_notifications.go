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
	"time"

	"github.com/riverqueue/river"
	"go.uber.org/zap"

	"github.com/ajbeattie/octobud/backend/internal/db"
	"github.com/ajbeattie/octobud/backend/internal/sync"
)

// SyncOlderNotificationsArgs are the arguments for the SyncOlderNotifications job.
// This job syncs notifications older than the current oldest synced notification.
type SyncOlderNotificationsArgs struct {
	// Days is the number of days to sync back from UntilTime
	Days int `json:"days"`
	// UntilTime is the cutoff - only sync notifications older than this
	// Typically set to oldest_notification_synced_at
	UntilTime time.Time `json:"untilTime"`
	// MaxCount is an optional limit on the number of notifications to sync
	MaxCount *int `json:"maxCount,omitempty"`
	// UnreadOnly filters to only sync unread notifications
	UnreadOnly bool `json:"unreadOnly"`
}

// Kind returns the unique identifier for this job type.
func (SyncOlderNotificationsArgs) Kind() string { return "sync_older_notifications" }

// InsertOpts specifies the queue or other options to use for the job.
func (SyncOlderNotificationsArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue: "sync_notifications", // Share queue with regular sync
	}
}

// SyncOlderNotificationsWorker handles syncing older notifications from GitHub.
// This job fetches notifications from a specific time range and queues processing jobs.
type SyncOlderNotificationsWorker struct {
	river.WorkerDefaults[SyncOlderNotificationsArgs]
	logger      *zap.Logger
	syncService sync.SyncOperations
	riverClient db.RiverClient
}

// NewSyncOlderNotificationsWorker creates a new SyncOlderNotificationsWorker.
func NewSyncOlderNotificationsWorker(
	logger *zap.Logger,
	syncService sync.SyncOperations,
	client db.RiverClient,
) *SyncOlderNotificationsWorker {
	return &SyncOlderNotificationsWorker{
		logger:      logger,
		syncService: syncService,
		riverClient: client,
	}
}

// Work executes a sync operation for older notifications.
func (w *SyncOlderNotificationsWorker) Work(
	ctx context.Context,
	job *river.Job[SyncOlderNotificationsArgs],
) error {
	args := job.Args

	// Compute the time range
	since := args.UntilTime.AddDate(0, 0, -args.Days)

	w.logger.Info("syncing older notifications",
		zap.Int64("jobID", job.ID),
		zap.Time("since", since),
		zap.Time("until", args.UntilTime),
		zap.Int("days", args.Days),
		zap.Any("maxCount", args.MaxCount),
		zap.Bool("unreadOnly", args.UnreadOnly))

	// Fetch older notifications using the sync service
	// The service handles GitHub API calls and filtering
	threads, err := w.syncService.FetchOlderNotificationsToSync(
		ctx,
		since,
		args.UntilTime,
		args.MaxCount,
		args.UnreadOnly,
	)
	if err != nil {
		w.logger.Error("failed to fetch older notifications",
			zap.Int64("jobID", job.ID),
			zap.Error(err))
		return err
	}

	if len(threads) == 0 {
		w.logger.Info("no older notifications found in time range",
			zap.Int64("jobID", job.ID))
		return nil
	}

	w.logger.Info("queuing older notifications for processing",
		zap.Int64("jobID", job.ID),
		zap.Int("count", len(threads)))

	// Track the oldest notification for updating sync state
	var oldestNotification time.Time

	// Queue individual processing jobs for each notification
	for _, thread := range threads {
		threadData, err := json.Marshal(thread)
		if err != nil {
			w.logger.Warn("failed to marshal notification thread",
				zap.Int64("jobID", job.ID),
				zap.String("threadID", thread.ID),
				zap.Error(err))
			continue
		}

		_, err = w.riverClient.Insert(ctx, ProcessNotificationArgs{
			NotificationData: threadData,
		}, nil)

		if err != nil {
			w.logger.Warn("failed to queue notification processing job",
				zap.Int64("jobID", job.ID),
				zap.String("threadID", thread.ID),
				zap.Error(err))
			continue
		}

		// Track oldest notification
		if oldestNotification.IsZero() || thread.UpdatedAt.Before(oldestNotification) {
			oldestNotification = thread.UpdatedAt
		}
	}

	// Update oldest_notification_synced_at if we found older notifications
	if !oldestNotification.IsZero() {
		w.logger.Info("updating oldest notification synced timestamp",
			zap.Int64("jobID", job.ID),
			zap.Time("oldestNotification", oldestNotification))

		// Use UpdateSyncStateAfterProcessingWithInitialSync to update oldest_notification_synced_at
		// Pass nil for initialSyncCompletedAt (don't change it) and the new oldest timestamp
		if err := w.syncService.UpdateSyncStateAfterProcessingWithInitialSync(
			ctx,
			time.Time{}, // Don't update latest_notification_at
			nil,         // Don't change initial_sync_completed_at
			&oldestNotification,
		); err != nil {
			w.logger.Warn("failed to update oldest notification timestamp",
				zap.Int64("jobID", job.ID),
				zap.Error(err))
		}
	}

	return nil
}
