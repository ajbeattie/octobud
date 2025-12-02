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

package user

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ajbeattie/octobud/backend/internal/api/auth"
	authmocks "github.com/ajbeattie/octobud/backend/internal/core/auth/mocks"
	syncstatemocks "github.com/ajbeattie/octobud/backend/internal/core/syncstate/mocks"
	dbmocks "github.com/ajbeattie/octobud/backend/internal/db/mocks"
	"github.com/ajbeattie/octobud/backend/internal/jobs"
	"github.com/ajbeattie/octobud/backend/internal/models"
)

func TestHandler_HandleGetSyncSettings(t *testing.T) {
	tests := []struct {
		name           string
		setupContext   func(*http.Request) *http.Request
		setupMock      func(*authmocks.MockAuthService)
		expectedStatus int
		expectedBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "success returns sync settings",
			setupContext: func(req *http.Request) *http.Request {
				ctx := auth.SetUsernameInContext(req.Context(), "admin")
				return req.WithContext(ctx)
			},
			setupMock: func(m *authmocks.MockAuthService) {
				days := 30
				maxCount := 1000
				m.EXPECT().GetUserSyncSettings(gomock.Any()).Return(&models.SyncSettings{
					InitialSyncDays:       &days,
					InitialSyncMaxCount:   &maxCount,
					InitialSyncUnreadOnly: true,
					SetupCompleted:        true,
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response SyncSettingsResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.NotNil(t, response.InitialSyncDays)
				require.Equal(t, 30, *response.InitialSyncDays)
				require.NotNil(t, response.InitialSyncMaxCount)
				require.Equal(t, 1000, *response.InitialSyncMaxCount)
				require.True(t, response.InitialSyncUnreadOnly)
				require.True(t, response.SetupCompleted)
			},
		},
		{
			name: "returns empty response when no settings exist",
			setupContext: func(req *http.Request) *http.Request {
				ctx := auth.SetUsernameInContext(req.Context(), "admin")
				return req.WithContext(ctx)
			},
			setupMock: func(m *authmocks.MockAuthService) {
				m.EXPECT().GetUserSyncSettings(gomock.Any()).Return(nil, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response SyncSettingsResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Nil(t, response.InitialSyncDays)
				require.Nil(t, response.InitialSyncMaxCount)
				require.False(t, response.InitialSyncUnreadOnly)
				require.False(t, response.SetupCompleted)
			},
		},
		{
			name: "missing username in context returns 401",
			setupContext: func(req *http.Request) *http.Request {
				return req
			},
			setupMock:      nil,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "service error returns 500",
			setupContext: func(req *http.Request) *http.Request {
				ctx := auth.SetUsernameInContext(req.Context(), "admin")
				return req.WithContext(ctx)
			},
			setupMock: func(m *authmocks.MockAuthService) {
				m.EXPECT().
					GetUserSyncSettings(gomock.Any()).
					Return(nil, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			handler, mockService := setupTestHandler(ctrl)
			if tt.setupMock != nil {
				tt.setupMock(mockService)
			}

			req := createRequest(http.MethodGet, "/api/user/sync-settings", nil)
			req = tt.setupContext(req)
			w := httptest.NewRecorder()

			handler.HandleGetSyncSettings(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != nil {
				tt.expectedBody(t, w)
			}
		})
	}
}

func TestHandler_HandleUpdateSyncSettings(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupContext   func(*http.Request) *http.Request
		setupMock      func(*authmocks.MockAuthService)
		expectedStatus int
		expectedBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "success updates all settings",
			requestBody: SyncSettingsRequest{
				InitialSyncDays:       intPtr(30),
				InitialSyncMaxCount:   intPtr(1000),
				InitialSyncUnreadOnly: true,
				SetupCompleted:        false,
			},
			setupContext: func(req *http.Request) *http.Request {
				ctx := auth.SetUsernameInContext(req.Context(), "admin")
				return req.WithContext(ctx)
			},
			setupMock: func(m *authmocks.MockAuthService) {
				m.EXPECT().GetUserSyncSettings(gomock.Any()).Return(nil, nil)
				m.EXPECT().UpdateUserSyncSettings(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, settings *models.SyncSettings) error {
						require.NotNil(t, settings.InitialSyncDays)
						require.Equal(t, 30, *settings.InitialSyncDays)
						require.NotNil(t, settings.InitialSyncMaxCount)
						require.Equal(t, 1000, *settings.InitialSyncMaxCount)
						require.True(t, settings.InitialSyncUnreadOnly)
						require.False(t, settings.SetupCompleted)
						return nil
					})
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response SyncSettingsResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.NotNil(t, response.InitialSyncDays)
				require.Equal(t, 30, *response.InitialSyncDays)
				require.NotNil(t, response.InitialSyncMaxCount)
				require.Equal(t, 1000, *response.InitialSyncMaxCount)
				require.True(t, response.InitialSyncUnreadOnly)
			},
		},
		{
			name: "success with nil days (all-time sync)",
			requestBody: SyncSettingsRequest{
				InitialSyncDays:       nil,
				InitialSyncMaxCount:   intPtr(500),
				InitialSyncUnreadOnly: false,
				SetupCompleted:        true,
			},
			setupContext: func(req *http.Request) *http.Request {
				ctx := auth.SetUsernameInContext(req.Context(), "admin")
				return req.WithContext(ctx)
			},
			setupMock: func(m *authmocks.MockAuthService) {
				m.EXPECT().GetUserSyncSettings(gomock.Any()).Return(nil, nil)
				m.EXPECT().UpdateUserSyncSettings(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, settings *models.SyncSettings) error {
						require.Nil(t, settings.InitialSyncDays)
						return nil
					})
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "missing username in context returns 401",
			requestBody: SyncSettingsRequest{
				InitialSyncDays: intPtr(30),
			},
			setupContext: func(req *http.Request) *http.Request {
				return req
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:        "invalid request body returns 400",
			requestBody: "invalid json",
			setupContext: func(req *http.Request) *http.Request {
				ctx := auth.SetUsernameInContext(req.Context(), "admin")
				return req.WithContext(ctx)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "initialSyncDays less than 1 returns 400",
			requestBody: SyncSettingsRequest{
				InitialSyncDays: intPtr(0),
			},
			setupContext: func(req *http.Request) *http.Request {
				ctx := auth.SetUsernameInContext(req.Context(), "admin")
				return req.WithContext(ctx)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Contains(t, response.Error, "initialSyncDays must be at least 1")
			},
		},
		{
			name: "initialSyncDays exceeds max returns 400",
			requestBody: SyncSettingsRequest{
				InitialSyncDays: intPtr(3651),
			},
			setupContext: func(req *http.Request) *http.Request {
				ctx := auth.SetUsernameInContext(req.Context(), "admin")
				return req.WithContext(ctx)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Contains(t, response.Error, "initialSyncDays cannot exceed 3650")
			},
		},
		{
			name: "initialSyncMaxCount less than 1 returns 400",
			requestBody: SyncSettingsRequest{
				InitialSyncMaxCount: intPtr(0),
			},
			setupContext: func(req *http.Request) *http.Request {
				ctx := auth.SetUsernameInContext(req.Context(), "admin")
				return req.WithContext(ctx)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Contains(t, response.Error, "initialSyncMaxCount must be at least 1")
			},
		},
		{
			name: "initialSyncMaxCount exceeds max returns 400",
			requestBody: SyncSettingsRequest{
				InitialSyncMaxCount: intPtr(100001),
			},
			setupContext: func(req *http.Request) *http.Request {
				ctx := auth.SetUsernameInContext(req.Context(), "admin")
				return req.WithContext(ctx)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Contains(t, response.Error, "initialSyncMaxCount cannot exceed 100000")
			},
		},
		{
			name: "service error returns 500",
			requestBody: SyncSettingsRequest{
				InitialSyncDays: intPtr(30),
			},
			setupContext: func(req *http.Request) *http.Request {
				ctx := auth.SetUsernameInContext(req.Context(), "admin")
				return req.WithContext(ctx)
			},
			setupMock: func(m *authmocks.MockAuthService) {
				m.EXPECT().GetUserSyncSettings(gomock.Any()).Return(nil, nil)
				m.EXPECT().
					UpdateUserSyncSettings(gomock.Any(), gomock.Any()).
					Return(errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			handler, mockService := setupTestHandler(ctrl)
			if tt.setupMock != nil {
				tt.setupMock(mockService)
			}

			req := createRequest(http.MethodPut, "/api/user/sync-settings", tt.requestBody)
			req = tt.setupContext(req)
			w := httptest.NewRecorder()

			handler.HandleUpdateSyncSettings(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != nil {
				tt.expectedBody(t, w)
			}
		})
	}
}

func TestHandler_HandleGetSyncState(t *testing.T) {
	tests := []struct {
		name           string
		setupContext   func(*http.Request) *http.Request
		setupHandler   func(*Handler, *gomock.Controller)
		expectedStatus int
		expectedBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "success returns sync state with timestamps",
			setupContext: func(req *http.Request) *http.Request {
				ctx := auth.SetUsernameInContext(req.Context(), "admin")
				return req.WithContext(ctx)
			},
			setupHandler: func(h *Handler, ctrl *gomock.Controller) {
				mockSyncState := syncstatemocks.NewMockSyncStateService(ctrl)
				oldestTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
				completedTime := time.Date(2024, 1, 20, 12, 0, 0, 0, time.UTC)
				mockSyncState.EXPECT().GetSyncState(gomock.Any()).Return(models.SyncState{
					OldestNotificationSyncedAt: sql.NullTime{Time: oldestTime, Valid: true},
					InitialSyncCompletedAt:     sql.NullTime{Time: completedTime, Valid: true},
				}, nil)
				h.syncStateSvc = mockSyncState
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response SyncStateResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.NotNil(t, response.OldestNotificationSyncedAt)
				require.NotNil(t, response.InitialSyncCompletedAt)
				require.Contains(t, *response.OldestNotificationSyncedAt, "2024-01-15")
				require.Contains(t, *response.InitialSyncCompletedAt, "2024-01-20")
			},
		},
		{
			name: "success returns empty response when no timestamps",
			setupContext: func(req *http.Request) *http.Request {
				ctx := auth.SetUsernameInContext(req.Context(), "admin")
				return req.WithContext(ctx)
			},
			setupHandler: func(h *Handler, ctrl *gomock.Controller) {
				mockSyncState := syncstatemocks.NewMockSyncStateService(ctrl)
				mockSyncState.EXPECT().GetSyncState(gomock.Any()).Return(models.SyncState{}, nil)
				h.syncStateSvc = mockSyncState
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response SyncStateResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Nil(t, response.OldestNotificationSyncedAt)
				require.Nil(t, response.InitialSyncCompletedAt)
			},
		},
		{
			name: "missing username returns 401",
			setupContext: func(req *http.Request) *http.Request {
				return req
			},
			setupHandler:   nil,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "sync state service unavailable returns 503",
			setupContext: func(req *http.Request) *http.Request {
				ctx := auth.SetUsernameInContext(req.Context(), "admin")
				return req.WithContext(ctx)
			},
			setupHandler: func(h *Handler, _ *gomock.Controller) {
				h.syncStateSvc = nil
			},
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name: "service error returns 500",
			setupContext: func(req *http.Request) *http.Request {
				ctx := auth.SetUsernameInContext(req.Context(), "admin")
				return req.WithContext(ctx)
			},
			setupHandler: func(h *Handler, ctrl *gomock.Controller) {
				mockSyncState := syncstatemocks.NewMockSyncStateService(ctrl)
				mockSyncState.EXPECT().
					GetSyncState(gomock.Any()).
					Return(models.SyncState{}, errors.New("database error"))
				h.syncStateSvc = mockSyncState
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			handler, _ := setupTestHandler(ctrl)
			if tt.setupHandler != nil {
				tt.setupHandler(handler, ctrl)
			}

			req := createRequest(http.MethodGet, "/api/user/sync-state", nil)
			req = tt.setupContext(req)
			w := httptest.NewRecorder()

			handler.HandleGetSyncState(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != nil {
				tt.expectedBody(t, w)
			}
		})
	}
}

func TestHandler_HandleSyncOlder(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupContext   func(*http.Request) *http.Request
		setupHandler   func(*Handler, *gomock.Controller)
		expectedStatus int
		expectedBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "success queues sync older job",
			requestBody: SyncOlderRequest{
				Days:       30,
				MaxCount:   intPtr(100),
				UnreadOnly: false,
			},
			setupContext: func(req *http.Request) *http.Request {
				ctx := auth.SetUsernameInContext(req.Context(), "admin")
				return req.WithContext(ctx)
			},
			setupHandler: func(h *Handler, ctrl *gomock.Controller) {
				oldestTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
				mockSyncState := syncstatemocks.NewMockSyncStateService(ctrl)
				mockSyncState.EXPECT().GetSyncState(gomock.Any()).Return(models.SyncState{
					OldestNotificationSyncedAt: sql.NullTime{Time: oldestTime, Valid: true},
				}, nil)
				h.syncStateSvc = mockSyncState

				mockRiver := dbmocks.NewMockRiverClient(ctrl)
				mockRiver.EXPECT().Insert(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, args jobs.SyncOlderNotificationsArgs, _ *river.InsertOpts) (*rivertype.JobInsertResult, error) {
						require.Equal(t, 30, args.Days)
						require.Equal(t, oldestTime, args.UntilTime)
						require.NotNil(t, args.MaxCount)
						require.Equal(t, 100, *args.MaxCount)
						require.False(t, args.UnreadOnly)
						return &rivertype.JobInsertResult{}, nil
					})
				h.riverClient = mockRiver
			},
			expectedStatus: http.StatusAccepted,
		},
		{
			name: "success with minimal request",
			requestBody: SyncOlderRequest{
				Days: 60,
			},
			setupContext: func(req *http.Request) *http.Request {
				ctx := auth.SetUsernameInContext(req.Context(), "admin")
				return req.WithContext(ctx)
			},
			setupHandler: func(h *Handler, ctrl *gomock.Controller) {
				oldestTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
				mockSyncState := syncstatemocks.NewMockSyncStateService(ctrl)
				mockSyncState.EXPECT().GetSyncState(gomock.Any()).Return(models.SyncState{
					OldestNotificationSyncedAt: sql.NullTime{Time: oldestTime, Valid: true},
				}, nil)
				h.syncStateSvc = mockSyncState

				mockRiver := dbmocks.NewMockRiverClient(ctrl)
				mockRiver.EXPECT().
					Insert(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&rivertype.JobInsertResult{}, nil)
				h.riverClient = mockRiver
			},
			expectedStatus: http.StatusAccepted,
		},
		{
			name: "missing username returns 401",
			requestBody: SyncOlderRequest{
				Days: 30,
			},
			setupContext: func(req *http.Request) *http.Request {
				return req
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:        "invalid request body returns 400",
			requestBody: "invalid json",
			setupContext: func(req *http.Request) *http.Request {
				ctx := auth.SetUsernameInContext(req.Context(), "admin")
				return req.WithContext(ctx)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "days less than 1 returns 400",
			requestBody: SyncOlderRequest{
				Days: 0,
			},
			setupContext: func(req *http.Request) *http.Request {
				ctx := auth.SetUsernameInContext(req.Context(), "admin")
				return req.WithContext(ctx)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Contains(t, response.Error, "days must be at least 1")
			},
		},
		{
			name: "days exceeds max returns 400",
			requestBody: SyncOlderRequest{
				Days: 3651,
			},
			setupContext: func(req *http.Request) *http.Request {
				ctx := auth.SetUsernameInContext(req.Context(), "admin")
				return req.WithContext(ctx)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Contains(t, response.Error, "days cannot exceed 3650")
			},
		},
		{
			name: "maxCount less than 1 returns 400",
			requestBody: SyncOlderRequest{
				Days:     30,
				MaxCount: intPtr(0),
			},
			setupContext: func(req *http.Request) *http.Request {
				ctx := auth.SetUsernameInContext(req.Context(), "admin")
				return req.WithContext(ctx)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Contains(t, response.Error, "maxCount must be at least 1")
			},
		},
		{
			name: "maxCount exceeds max returns 400",
			requestBody: SyncOlderRequest{
				Days:     30,
				MaxCount: intPtr(100001),
			},
			setupContext: func(req *http.Request) *http.Request {
				ctx := auth.SetUsernameInContext(req.Context(), "admin")
				return req.WithContext(ctx)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Contains(t, response.Error, "maxCount cannot exceed 100000")
			},
		},
		{
			name: "sync state service unavailable returns 503",
			requestBody: SyncOlderRequest{
				Days: 30,
			},
			setupContext: func(req *http.Request) *http.Request {
				ctx := auth.SetUsernameInContext(req.Context(), "admin")
				return req.WithContext(ctx)
			},
			setupHandler: func(h *Handler, _ *gomock.Controller) {
				h.syncStateSvc = nil
			},
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name: "river client unavailable returns 503",
			requestBody: SyncOlderRequest{
				Days: 30,
			},
			setupContext: func(req *http.Request) *http.Request {
				ctx := auth.SetUsernameInContext(req.Context(), "admin")
				return req.WithContext(ctx)
			},
			setupHandler: func(h *Handler, ctrl *gomock.Controller) {
				mockSyncState := syncstatemocks.NewMockSyncStateService(ctrl)
				h.syncStateSvc = mockSyncState
				h.riverClient = nil
			},
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name: "no oldest notification returns 400",
			requestBody: SyncOlderRequest{
				Days: 30,
			},
			setupContext: func(req *http.Request) *http.Request {
				ctx := auth.SetUsernameInContext(req.Context(), "admin")
				return req.WithContext(ctx)
			},
			setupHandler: func(h *Handler, ctrl *gomock.Controller) {
				mockSyncState := syncstatemocks.NewMockSyncStateService(ctrl)
				mockSyncState.EXPECT().GetSyncState(gomock.Any()).Return(models.SyncState{
					OldestNotificationSyncedAt: sql.NullTime{Valid: false},
				}, nil)
				h.syncStateSvc = mockSyncState

				mockRiver := dbmocks.NewMockRiverClient(ctrl)
				h.riverClient = mockRiver
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Contains(t, response.Error, "No notifications have been synced yet")
			},
		},
		{
			name: "sync state error returns 500",
			requestBody: SyncOlderRequest{
				Days: 30,
			},
			setupContext: func(req *http.Request) *http.Request {
				ctx := auth.SetUsernameInContext(req.Context(), "admin")
				return req.WithContext(ctx)
			},
			setupHandler: func(h *Handler, ctrl *gomock.Controller) {
				mockSyncState := syncstatemocks.NewMockSyncStateService(ctrl)
				mockSyncState.EXPECT().
					GetSyncState(gomock.Any()).
					Return(models.SyncState{}, errors.New("database error"))
				h.syncStateSvc = mockSyncState

				mockRiver := dbmocks.NewMockRiverClient(ctrl)
				h.riverClient = mockRiver
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "job queue error returns 500",
			requestBody: SyncOlderRequest{
				Days: 30,
			},
			setupContext: func(req *http.Request) *http.Request {
				ctx := auth.SetUsernameInContext(req.Context(), "admin")
				return req.WithContext(ctx)
			},
			setupHandler: func(h *Handler, ctrl *gomock.Controller) {
				oldestTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
				mockSyncState := syncstatemocks.NewMockSyncStateService(ctrl)
				mockSyncState.EXPECT().GetSyncState(gomock.Any()).Return(models.SyncState{
					OldestNotificationSyncedAt: sql.NullTime{Time: oldestTime, Valid: true},
				}, nil)
				h.syncStateSvc = mockSyncState

				mockRiver := dbmocks.NewMockRiverClient(ctrl)
				mockRiver.EXPECT().
					Insert(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New("queue error"))
				h.riverClient = mockRiver
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			handler, _ := setupTestHandler(ctrl)
			if tt.setupHandler != nil {
				tt.setupHandler(handler, ctrl)
			}

			req := createRequest(http.MethodPost, "/api/user/sync-older", tt.requestBody)
			req = tt.setupContext(req)
			w := httptest.NewRecorder()

			handler.HandleSyncOlder(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != nil {
				tt.expectedBody(t, w)
			}
		})
	}
}
