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

// Package interfaces defines interfaces for GitHub operations.
package interfaces //nolint:revive // Sigh.

import (
	"context"
	"time"

	"github.com/ajbeattie/octobud/backend/internal/github/types"
)

// SyncOperations interface defines operations for syncing notifications
type SyncOperations interface {
	FetchNotificationsToSync(ctx context.Context) ([]types.NotificationThread, error)
	UpdateSyncStateAfterProcessing(ctx context.Context, latestUpdate time.Time) error
	ProcessNotification(ctx context.Context, thread types.NotificationThread) error
}
