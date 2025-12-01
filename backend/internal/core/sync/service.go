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

// Package sync provides the sync service.
package sync

import (
	"context"
	"time"

	"github.com/ajbeattie/octobud/backend/internal/db"
	"github.com/ajbeattie/octobud/backend/internal/models"
)

// SyncService is the interface for the sync service.
//
//nolint:revive // exported type name stutters with package name
type SyncService interface {
	GetSyncState(ctx context.Context) (models.SyncState, error)
	UpsertSyncState(
		ctx context.Context,
		lastSuccessfulPoll *time.Time,
		latestNotificationAt *time.Time,
	) (models.SyncState, error)
}

// Service provides business logic for sync operations
type Service struct {
	queries db.Store
}

// NewService constructs a Service backed by the provided queries
func NewService(queries db.Store) *Service {
	return &Service{
		queries: queries,
	}
}
