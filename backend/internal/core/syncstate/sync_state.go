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

package syncstate

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/ajbeattie/octobud/backend/internal/db"
	"github.com/ajbeattie/octobud/backend/internal/models"
)

// Error definitions
var (
	ErrFailedToGetSyncState    = errors.New("failed to get sync state")
	ErrFailedToUpdateSyncState = errors.New("failed to update sync state")
)

// GetSyncState returns the current sync state
func (s *Service) GetSyncState(ctx context.Context) (models.SyncState, error) {
	state, err := s.queries.GetSyncState(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.SyncState{}, nil
		}
		return models.SyncState{}, errors.Join(ErrFailedToGetSyncState, err)
	}

	return models.SyncState{
		LastSuccessfulPoll:         state.LastSuccessfulPoll,
		LatestNotificationAt:       state.LatestNotificationAt,
		UpdatedAt:                  state.UpdatedAt,
		InitialSyncCompletedAt:     state.InitialSyncCompletedAt,
		OldestNotificationSyncedAt: state.OldestNotificationSyncedAt,
	}, nil
}

// UpsertSyncState updates or creates the sync state
func (s *Service) UpsertSyncState(
	ctx context.Context,
	lastSuccessfulPoll, latestNotificationAt *time.Time,
) (models.SyncState, error) {
	return s.UpsertSyncStateWithInitialSync(ctx, lastSuccessfulPoll, latestNotificationAt, nil, nil)
}

// UpsertSyncStateWithInitialSync updates or creates the sync state with initial sync tracking
func (s *Service) UpsertSyncStateWithInitialSync(
	ctx context.Context,
	lastSuccessfulPoll, latestNotificationAt *time.Time,
	initialSyncCompletedAt, oldestNotificationSyncedAt *time.Time,
) (models.SyncState, error) {
	params := db.UpsertSyncStateParams{
		LastSuccessfulPoll:         models.SQLNullTime(lastSuccessfulPoll),
		LatestNotificationAt:       models.SQLNullTime(latestNotificationAt),
		LastNotificationEtag:       sql.NullString{},
		InitialSyncCompletedAt:     models.SQLNullTime(initialSyncCompletedAt),
		OldestNotificationSyncedAt: models.SQLNullTime(oldestNotificationSyncedAt),
	}

	result, err := s.queries.UpsertSyncState(ctx, params)
	if err != nil {
		return models.SyncState{}, errors.Join(ErrFailedToUpdateSyncState, err)
	}

	return models.SyncState{
		LastSuccessfulPoll:         result.LastSuccessfulPoll,
		LatestNotificationAt:       result.LatestNotificationAt,
		UpdatedAt:                  result.UpdatedAt,
		InitialSyncCompletedAt:     result.InitialSyncCompletedAt,
		OldestNotificationSyncedAt: result.OldestNotificationSyncedAt,
	}, nil
}
