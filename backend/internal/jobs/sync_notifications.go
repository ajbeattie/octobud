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

// SyncNotificationsArgs are the arguments for the SyncNotifications job.
type SyncNotificationsArgs struct{}

// Kind returns the unique identifier for this job type.
func (SyncNotificationsArgs) Kind() string { return "sync_notifications" }

// InsertOpts specifies the queue or other options to use for the job.
func (SyncNotificationsArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue: "sync_notifications",
	}
}

// SyncNotificationsWorker handles syncing notifications from GitHub.
// This job fetches notifications from GitHub and queues individual ProcessNotification jobs.
type SyncNotificationsWorker struct {
	river.WorkerDefaults[SyncNotificationsArgs]
	logger      *zap.Logger
	syncService sync.SyncOperations
	riverClient db.RiverClient
}

// NewSyncNotificationsWorker creates a new SyncNotificationsWorker.
func NewSyncNotificationsWorker(
	logger *zap.Logger,
	syncService sync.SyncOperations,
	client db.RiverClient,
) *SyncNotificationsWorker {
	return &SyncNotificationsWorker{
		logger:      logger,
		syncService: syncService,
		riverClient: client,
	}
}

// Work executes a single sync operation by fetching notifications and queuing processing jobs.
func (w *SyncNotificationsWorker) Work(
	ctx context.Context,
	job *river.Job[SyncNotificationsArgs],
) error {
	// Get all sync context at the start - single source of truth
	syncCtx, err := w.syncService.GetSyncContext(ctx)
	if err != nil {
		w.logger.Warn("failed to get sync context",
			zap.Int64("jobID", job.ID),
			zap.Error(err))
		return nil
	}

	// If sync isn't configured yet, skip entirely
	if !syncCtx.IsSyncConfigured {
		w.logger.Debug("sync not configured yet, skipping", zap.Int64("jobID", job.ID))
		return nil
	}

	// Log sync status
	if syncCtx.IsInitialSync {
		if syncCtx.OldestNotificationSyncedAt.IsZero() {
			w.logger.Info("initial sync starting", zap.Int64("jobID", job.ID))
		} else {
			w.logger.Info("initial sync in progress",
				zap.Int64("jobID", job.ID),
				zap.Time("oldestSoFar", syncCtx.OldestNotificationSyncedAt))
		}
	}

	// Fetch notifications from GitHub using the pre-computed context
	threads, err := w.syncService.FetchNotificationsToSync(ctx, syncCtx)
	if err != nil {
		return err
	}

	// If this was an initial sync and we found zero notifications, mark as complete immediately
	// This handles the case where user has no notifications (empty account)
	if syncCtx.IsInitialSync && len(threads) == 0 {
		w.logger.Info("initial sync found 0 notifications, marking as complete",
			zap.Int64("jobID", job.ID))
		now := time.Now().UTC()
		var zeroTime time.Time // No latest notification time (empty account)
		if err := w.syncService.UpdateSyncStateAfterProcessingWithInitialSync(
			ctx,
			zeroTime, // No latest notification time
			&now,     // Mark initial sync as complete now
			nil,      // No oldest notification
		); err != nil {
			w.logger.Warn("failed to mark initial sync complete with 0 notifications",
				zap.Int64("jobID", job.ID),
				zap.Error(err))
		}
		return nil
	}

	if len(threads) == 0 {
		w.logger.Debug("no new notifications found", zap.Int64("jobID", job.ID))
		return nil
	}

	w.logger.Info("queuing notifications for processing",
		zap.Int64("jobID", job.ID),
		zap.Int("count", len(threads)))

	// Track the latest and oldest update times for sync state
	var latestUpdate time.Time
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

		if thread.UpdatedAt.After(latestUpdate) {
			latestUpdate = thread.UpdatedAt
		}

		// Track oldest notification for initial sync tracking
		if syncCtx.IsInitialSync {
			if oldestNotification.IsZero() || thread.UpdatedAt.Before(oldestNotification) {
				oldestNotification = thread.UpdatedAt
			}
		}
	}

	// Update sync state after queuing all jobs
	if !latestUpdate.IsZero() {
		if syncCtx.IsInitialSync {
			// Use the oldest from this batch, or fall back to existing oldest
			oldestToUse := oldestNotification
			if oldestToUse.IsZero() && !syncCtx.OldestNotificationSyncedAt.IsZero() {
				oldestToUse = syncCtx.OldestNotificationSyncedAt
			}

			// Mark as complete if we have an oldest notification
			if !oldestToUse.IsZero() {
				now := time.Now().UTC()
				w.logger.Info("marking initial sync as complete",
					zap.Int64("jobID", job.ID),
					zap.Time("oldestNotification", oldestToUse))
				if err := w.syncService.UpdateSyncStateAfterProcessingWithInitialSync(
					ctx,
					latestUpdate,
					&now,
					&oldestToUse,
				); err != nil {
					// Log but don't fail - jobs are already queued
					w.logger.Warn("failed to update sync state after initial sync",
						zap.Int64("jobID", job.ID),
						zap.Error(err))
				}
			}
		} else {
			// Regular update without initial sync tracking
			if err := w.syncService.UpdateSyncStateAfterProcessing(ctx, latestUpdate); err != nil {
				// Log but don't fail - jobs are already queued
				w.logger.Warn("failed to update sync state after processing",
					zap.Int64("jobID", job.ID),
					zap.Error(err))
			}
		}
	}

	return nil
}
