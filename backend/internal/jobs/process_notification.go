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

	"github.com/riverqueue/river"

	"github.com/ajbeattie/octobud/backend/internal/db"
	"github.com/ajbeattie/octobud/backend/internal/github/interfaces"
	"github.com/ajbeattie/octobud/backend/internal/github/types"
)

// ProcessNotificationArgs represents a single notification to process
type ProcessNotificationArgs struct {
	NotificationData json.RawMessage `json:"notification_data"`
}

// Kind returns the unique identifier for this job type.
func (ProcessNotificationArgs) Kind() string { return "process_notification" }

// InsertOpts specifies the queue or other options to use for the job.
func (ProcessNotificationArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue: "process_notification",
	}
}

// ProcessNotificationWorker handles processing of individual notifications
type ProcessNotificationWorker struct {
	river.WorkerDefaults[ProcessNotificationArgs]
	dbConn      *sql.DB
	queries     *db.Queries
	syncService interfaces.SyncOperations
}

// NewProcessNotificationWorker creates a new ProcessNotificationWorker.
func NewProcessNotificationWorker(
	dbConn *sql.DB,
	syncService interfaces.SyncOperations,
) *ProcessNotificationWorker {
	return &ProcessNotificationWorker{
		dbConn:      dbConn,
		queries:     db.New(dbConn),
		syncService: syncService,
	}
}

// Work processes a notification.
func (w *ProcessNotificationWorker) Work(
	ctx context.Context,
	job *river.Job[ProcessNotificationArgs],
) error {
	var thread types.NotificationThread
	if err := json.Unmarshal(job.Args.NotificationData, &thread); err != nil {
		return err
	}

	// Check if notification already exists (to determine if this is INSERT or UPDATE)
	var isNewNotification bool
	if w.dbConn != nil && w.queries != nil {
		existingNotification, err := w.queries.GetNotificationByGithubID(ctx, thread.ID)
		isNewNotification = err != nil // If we got an error, notification doesn't exist yet
		_ = existingNotification       // Suppress unused variable warning
	} else {
		// In tests or when db connection is not available, assume it's a new notification
		isNewNotification = true
	}

	if err := w.syncService.ProcessNotification(ctx, thread); err != nil {
		return err
	}

	// Only apply rules to newly created notifications (INSERT), not updates
	// Skip rule application if db connection is not available (e.g., in tests)
	if isNewNotification && w.dbConn != nil && w.queries != nil {
		notification, err := w.queries.GetNotificationByGithubID(ctx, thread.ID)
		if err == nil {
			matcher := NewRuleMatcher(w.queries)
			_, matchErr := matcher.MatchAndApplyRules(ctx, notification.ID)
			if matchErr != nil {
				// Log the error but don't fail the job - rule application is best-effort
				return nil
			}
		}
	}

	return nil
}
