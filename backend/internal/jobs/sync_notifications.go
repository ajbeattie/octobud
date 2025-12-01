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
	"log"
	"time"

	"github.com/riverqueue/river"

	"github.com/ajbeattie/octobud/backend/internal/db"
	"github.com/ajbeattie/octobud/backend/internal/github/interfaces"
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
	syncService interfaces.SyncOperations
	riverClient db.RiverClient
}

// NewSyncNotificationsWorker creates a new SyncNotificationsWorker.
func NewSyncNotificationsWorker(
	syncService interfaces.SyncOperations,
	client db.RiverClient,
) *SyncNotificationsWorker {
	return &SyncNotificationsWorker{
		syncService: syncService,
		riverClient: client,
	}
}

// Work executes a single sync operation by fetching notifications and queuing processing jobs.
func (w *SyncNotificationsWorker) Work(
	ctx context.Context,
	job *river.Job[SyncNotificationsArgs],
) error {
	// Fetch notifications from GitHub
	threads, err := w.syncService.FetchNotificationsToSync(ctx)
	if err != nil {
		return err
	}

	if len(threads) == 0 {
		return nil
	}

	log.Printf("INFO: SyncNotifications job=%d queuing %d notification(s)", job.ID, len(threads))

	// Track the latest update time for sync state
	var latestUpdate time.Time

	// Queue individual processing jobs for each notification
	for _, thread := range threads {
		threadData, err := json.Marshal(thread)
		if err != nil {
			continue
		}

		_, err = w.riverClient.Insert(ctx, ProcessNotificationArgs{
			NotificationData: threadData,
		}, nil)

		if err != nil {
			continue
		}

		if thread.UpdatedAt.After(latestUpdate) {
			latestUpdate = thread.UpdatedAt
		}
	}

	// Update sync state after queuing all jobs
	if !latestUpdate.IsZero() {
		if err := w.syncService.UpdateSyncStateAfterProcessing(ctx, latestUpdate); err != nil {
			// Don't return error here - jobs are already queued
			_ = err // explicitly ignore error
		}
	}

	return nil
}
