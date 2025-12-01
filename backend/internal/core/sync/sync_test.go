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

package sync

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ajbeattie/octobud/backend/internal/db"
	"github.com/ajbeattie/octobud/backend/internal/db/mocks"
	"github.com/ajbeattie/octobud/backend/internal/models"
)

func TestService_GetSyncState(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(*mocks.MockStore)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, models.SyncState)
	}{
		{
			name: "success returns sync state",
			setupMock: func(m *mocks.MockStore) {
				now := time.Now().UTC()
				expectedState := db.GetSyncStateRow{
					ID:                   1,
					LastSuccessfulPoll:   sql.NullTime{Time: now, Valid: true},
					LatestNotificationAt: sql.NullTime{Time: now, Valid: true},
					UpdatedAt:            now,
				}
				m.EXPECT().
					GetSyncState(gomock.Any()).
					Return(expectedState, nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, state models.SyncState) {
				require.True(t, state.LastSuccessfulPoll.Valid)
				require.True(t, state.LatestNotificationAt.Valid)
			},
		},
		{
			name: "no rows returns empty state without error",
			setupMock: func(m *mocks.MockStore) {
				m.EXPECT().
					GetSyncState(gomock.Any()).
					Return(db.GetSyncStateRow{}, sql.ErrNoRows)
			},
			expectErr: false,
			checkResult: func(t *testing.T, state models.SyncState) {
				require.False(t, state.LastSuccessfulPoll.Valid)
				require.False(t, state.LatestNotificationAt.Valid)
			},
		},
		{
			name: "error wrapping database failure",
			setupMock: func(m *mocks.MockStore) {
				dbError := errors.New("database connection failed")
				m.EXPECT().
					GetSyncState(gomock.Any()).
					Return(db.GetSyncStateRow{}, dbError)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrFailedToGetSyncState))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.GetSyncState(ctx)

			if tt.expectErr {
				require.Error(t, err)
				if tt.checkErr != nil {
					tt.checkErr(t, err)
				}
			} else {
				require.NoError(t, err)
				if tt.checkResult != nil {
					tt.checkResult(t, result)
				}
			}
		})
	}
}
