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

package notification

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
)

func TestService_GetByGithubID(t *testing.T) {
	tests := []struct {
		name        string
		githubID    string
		setupMock   func(*mocks.MockStore, string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, db.Notification)
	}{
		{
			name:     "success returns notification",
			githubID: "abc",
			setupMock: func(m *mocks.MockStore, id string) {
				now := time.Now().UTC()
				expectedNotification := db.Notification{
					ID:           1,
					GithubID:     id,
					RepositoryID: 10,
					ImportedAt:   now,
				}
				m.EXPECT().
					GetNotificationByGithubID(gomock.Any(), id).
					Return(expectedNotification, nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, notification db.Notification) {
				require.Equal(t, "abc", notification.GithubID)
			},
		},
		{
			name:     "error wrapping not found",
			githubID: "not-found",
			setupMock: func(m *mocks.MockStore, id string) {
				m.EXPECT().
					GetNotificationByGithubID(gomock.Any(), id).
					Return(db.Notification{}, sql.ErrNoRows)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.ErrorIs(t, err, sql.ErrNoRows)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, tt.githubID)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.GetByGithubID(ctx, tt.githubID)

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

func TestService_ListNotificationsFromQueryString(t *testing.T) {
	tests := []struct {
		name        string
		queryStr    string
		limit       int32
		setupMock   func(*mocks.MockStore, string, int32)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, []db.Notification)
	}{
		{
			name:     "success returns notifications",
			queryStr: "is:unread",
			limit:    10,
			setupMock: func(m *mocks.MockStore, _ string, _ int32) {
				expectedResult := db.ListNotificationsFromQueryResult{
					Notifications: []db.Notification{
						{ID: 1, GithubID: "notif-1"},
						{ID: 2, GithubID: "notif-2"},
					},
					Total: 2,
				}
				m.EXPECT().
					ListNotificationsFromQuery(gomock.Any(), gomock.Any()).
					Return(expectedResult, nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, notifications []db.Notification) {
				require.Len(t, notifications, 2)
			},
		},
		{
			name:     "error wrapping database failure",
			queryStr: "is:unread",
			limit:    10,
			setupMock: func(m *mocks.MockStore, _ string, _ int32) {
				queryError := errors.New("query build failed")
				m.EXPECT().
					ListNotificationsFromQuery(gomock.Any(), gomock.Any()).
					Return(db.ListNotificationsFromQueryResult{}, queryError)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrFailedToListNotifications))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, tt.queryStr, tt.limit)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.ListNotificationsFromQueryString(ctx, tt.queryStr, tt.limit)

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

func TestService_GetTagsForNotification(t *testing.T) {
	tests := []struct {
		name           string
		notificationID int64
		setupMock      func(*mocks.MockStore, int64)
		expectErr      bool
		checkErr       func(*testing.T, error)
		checkResult    func(*testing.T, []db.Tag)
	}{
		{
			name:           "success returns tags",
			notificationID: 1,
			setupMock: func(m *mocks.MockStore, _ int64) {
				expectedTags := []db.Tag{
					{ID: 1, Name: "bug", Slug: "bug"},
					{ID: 2, Name: "feature", Slug: "feature"},
				}
				m.EXPECT().
					ListTagsForEntity(gomock.Any(), gomock.Any()).
					Return(expectedTags, nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, tags []db.Tag) {
				require.Len(t, tags, 2)
				require.Equal(t, "bug", tags[0].Name)
			},
		},
		{
			name:           "no rows returns empty slice without error",
			notificationID: 1,
			setupMock: func(m *mocks.MockStore, _ int64) {
				// sql.ErrNoRows should not be wrapped as an error
				m.EXPECT().
					ListTagsForEntity(gomock.Any(), gomock.Any()).
					Return([]db.Tag{}, sql.ErrNoRows)
			},
			expectErr: false,
			checkResult: func(t *testing.T, tags []db.Tag) {
				require.Empty(t, tags)
			},
		},
		{
			name:           "error wrapping database failure",
			notificationID: 1,
			setupMock: func(m *mocks.MockStore, _ int64) {
				dbError := errors.New("database query failed")
				m.EXPECT().
					ListTagsForEntity(gomock.Any(), gomock.Any()).
					Return([]db.Tag{}, dbError)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrFailedToFetchTags))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, tt.notificationID)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.GetTagsForNotification(ctx, tt.notificationID)

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

func TestService_BulkRemoveTag(t *testing.T) {
	tests := []struct {
		name          string
		notifications []db.Notification
		tagID         int64
		setupMock     func(*mocks.MockStore, []db.Notification, int64)
		expectErr     bool
		expectedCount int
		checkErr      func(*testing.T, error)
	}{
		{
			name: "success removes tag from notifications",
			notifications: []db.Notification{
				{ID: 1, GithubID: "notif-1"},
				{ID: 2, GithubID: "notif-2"},
			},
			tagID: 10,
			setupMock: func(m *mocks.MockStore, notifs []db.Notification, _ int64) {
				for _, notif := range notifs {
					m.EXPECT().
						RemoveTagAssignment(gomock.Any(), gomock.Any()).
						Return(nil)
					m.EXPECT().
						UpdateNotificationTagIds(gomock.Any(), notif.ID).
						Return(nil)
				}
			},
			expectErr:     false,
			expectedCount: 2,
		},
		{
			name: "error on remove tag assignment skips notification",
			notifications: []db.Notification{
				{ID: 1, GithubID: "notif-1"},
				{ID: 2, GithubID: "notif-2"},
			},
			tagID: 10,
			setupMock: func(m *mocks.MockStore, _ []db.Notification, _ int64) {
				// First notification fails
				m.EXPECT().
					RemoveTagAssignment(gomock.Any(), gomock.Any()).
					Return(errors.New("database error"))
				// Second notification succeeds
				m.EXPECT().
					RemoveTagAssignment(gomock.Any(), gomock.Any()).
					Return(nil)
				m.EXPECT().
					UpdateNotificationTagIds(gomock.Any(), int64(2)).
					Return(nil)
			},
			expectErr:     false,
			expectedCount: 1,
		},
		{
			name: "error on update tag IDs skips notification",
			notifications: []db.Notification{
				{ID: 1, GithubID: "notif-1"},
				{ID: 2, GithubID: "notif-2"},
			},
			tagID: 10,
			setupMock: func(m *mocks.MockStore, _ []db.Notification, _ int64) {
				// First notification: remove succeeds, update fails
				m.EXPECT().
					RemoveTagAssignment(gomock.Any(), gomock.Any()).
					Return(nil)
				m.EXPECT().
					UpdateNotificationTagIds(gomock.Any(), int64(1)).
					Return(errors.New("database error"))
				// Second notification succeeds
				m.EXPECT().
					RemoveTagAssignment(gomock.Any(), gomock.Any()).
					Return(nil)
				m.EXPECT().
					UpdateNotificationTagIds(gomock.Any(), int64(2)).
					Return(nil)
			},
			expectErr:     false,
			expectedCount: 1,
		},
		{
			name:          "empty notifications returns zero count",
			notifications: []db.Notification{},
			tagID:         10,
			setupMock: func(_ *mocks.MockStore, _ []db.Notification, _ int64) {
				// No mock expectations
			},
			expectErr:     false,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, tt.notifications, tt.tagID)
			service := NewService(mockQuerier)

			ctx := context.Background()
			count, err := service.BulkRemoveTag(ctx, tt.notifications, tt.tagID)

			if tt.expectErr {
				require.Error(t, err)
				if tt.checkErr != nil {
					tt.checkErr(t, err)
				}
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedCount, count)
			}
		})
	}
}

func TestService_NewEvaluator(t *testing.T) {
	tests := []struct {
		name      string
		queryStr  string
		expectErr bool
		checkErr  func(*testing.T, error)
	}{
		{
			name:      "success creates evaluator for valid query",
			queryStr:  "is:unread",
			expectErr: false,
		},
		{
			name:      "success creates evaluator for empty query",
			queryStr:  "",
			expectErr: false,
		},
		{
			name:      "success creates evaluator for complex query",
			queryStr:  "repo:cli/cli is:unread archived:false",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			service := NewService(mockQuerier)

			evaluator, err := service.NewEvaluator(tt.queryStr)

			if tt.expectErr {
				require.Error(t, err)
				require.Nil(t, evaluator)
				if tt.checkErr != nil {
					tt.checkErr(t, err)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, evaluator)
			}
		})
	}
}

func TestService_IndexRepositories(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(*mocks.MockStore)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, map[int64]db.Repository)
	}{
		{
			name: "success indexes repositories",
			setupMock: func(m *mocks.MockStore) {
				repositories := []db.Repository{
					{ID: 1, Name: "repo1", FullName: "owner/repo1"},
					{ID: 2, Name: "repo2", FullName: "owner/repo2"},
				}
				m.EXPECT().
					ListRepositories(gomock.Any()).
					Return(repositories, nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, repoMap map[int64]db.Repository) {
				require.Len(t, repoMap, 2)
				require.Equal(t, "repo1", repoMap[1].Name)
				require.Equal(t, "repo2", repoMap[2].Name)
			},
		},
		{
			name: "empty repositories returns empty map",
			setupMock: func(m *mocks.MockStore) {
				m.EXPECT().
					ListRepositories(gomock.Any()).
					Return([]db.Repository{}, nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, repoMap map[int64]db.Repository) {
				require.Empty(t, repoMap)
			},
		},
		{
			name: "error wrapping list repositories failure",
			setupMock: func(m *mocks.MockStore) {
				dbError := errors.New("database error")
				m.EXPECT().
					ListRepositories(gomock.Any()).
					Return(nil, dbError)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.Error(t, err)
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
			result, err := service.IndexRepositories(ctx)

			if tt.expectErr {
				require.Error(t, err)
				require.Nil(t, result)
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
