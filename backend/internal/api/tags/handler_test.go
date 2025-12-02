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

package tags

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/ajbeattie/octobud/backend/internal/core/tag"
	"github.com/ajbeattie/octobud/backend/internal/db"
	"github.com/ajbeattie/octobud/backend/internal/db/mocks"
)

func setupTestHandler(ctrl *gomock.Controller) (*Handler, *mocks.MockStore) {
	logger := zap.NewNop()
	mockStore := mocks.NewMockStore(ctrl)
	tagSvc := tag.NewService(mockStore)
	return New(logger, tagSvc), mockStore
}

func createRequest(method, url string, body interface{}) *http.Request {
	var reqBody []byte
	if body != nil {
		var err error
		reqBody, err = json.Marshal(body)
		if err != nil {
			// In test helper, return empty body on marshal error
			reqBody = nil
		}
	}
	req := httptest.NewRequest(method, url, bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestHandler_handleListAllTags(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*mocks.MockStore)
		expectedStatus int
		expectedBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "success returns list of tags",
			setupMock: func(m *mocks.MockStore) {
				tags := []db.Tag{
					{ID: 1, Name: "bug", Slug: "bug", DisplayOrder: 0},
					{ID: 2, Name: "feature", Slug: "feature", DisplayOrder: 1},
				}
				m.EXPECT().ListAllTags(gomock.Any()).Return(tags, nil)
				// Mock unread count queries for each tag
				m.EXPECT().ListNotificationsFromQuery(gomock.Any(), gomock.Any()).
					Return(db.ListNotificationsFromQueryResult{Total: 5}, nil).Times(2)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response listTagsResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Len(t, response.Tags, 2)
				require.Equal(t, "1", response.Tags[0].ID)
				require.Equal(t, "bug", response.Tags[0].Name)
			},
		},
		{
			name: "service error returns 500",
			setupMock: func(m *mocks.MockStore) {
				m.EXPECT().ListAllTags(gomock.Any()).Return(nil, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "failed to list tags", response.Error)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			handler, mockStore := setupTestHandler(ctrl)
			if tt.setupMock != nil {
				tt.setupMock(mockStore)
			}

			req := createRequest(http.MethodGet, "/tags", nil)
			w := httptest.NewRecorder()

			handler.handleListAllTags(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != nil {
				tt.expectedBody(t, w)
			}
		})
	}
}

func TestHandler_handleCreateTag(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*mocks.MockStore)
		expectedStatus int
		expectedBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "success creates tag",
			requestBody: createTagRequest{
				Name:        "Test Tag",
				Color:       stringPtr("#ff0000"),
				Description: stringPtr("Test description"),
			},
			setupMock: func(m *mocks.MockStore) {
				m.EXPECT().UpsertTag(gomock.Any(), gomock.Any()).Return(db.Tag{
					ID:           1,
					Name:         "Test Tag",
					Slug:         "test-tag",
					Color:        sql.NullString{String: "#ff0000", Valid: true},
					Description:  sql.NullString{String: "Test description", Valid: true},
					DisplayOrder: 0,
				}, nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response tagEnvelope
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "1", response.Tag.ID)
				require.Equal(t, "Test Tag", response.Tag.Name)
			},
		},
		{
			name:           "invalid body returns 400",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "invalid request body", response.Error)
			},
		},
		{
			name: "missing name returns 400",
			requestBody: createTagRequest{
				Name: "",
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "name is required", response.Error)
			},
		},
		{
			name: "service error returns 500",
			requestBody: createTagRequest{
				Name: "Test Tag",
			},
			setupMock: func(m *mocks.MockStore) {
				m.EXPECT().
					UpsertTag(gomock.Any(), gomock.Any()).
					Return(db.Tag{}, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "failed to create tag", response.Error)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			handler, mockStore := setupTestHandler(ctrl)
			if tt.setupMock != nil {
				tt.setupMock(mockStore)
			}

			req := createRequest(http.MethodPost, "/tags", tt.requestBody)
			w := httptest.NewRecorder()

			handler.handleCreateTag(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != nil {
				tt.expectedBody(t, w)
			}
		})
	}
}

func TestHandler_handleUpdateTag(t *testing.T) {
	tests := []struct {
		name           string
		tagID          string
		requestBody    interface{}
		setupMock      func(*mocks.MockStore, int64)
		expectedStatus int
		expectedBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:  "success updates tag",
			tagID: "1",
			requestBody: updateTagRequest{
				Name:        "Updated Tag",
				Color:       stringPtr("#00ff00"),
				Description: stringPtr("Updated description"),
			},
			setupMock: func(m *mocks.MockStore, tagID int64) {
				m.EXPECT().UpdateTag(gomock.Any(), gomock.Any()).Return(db.Tag{
					ID:           tagID,
					Name:         "Updated Tag",
					Slug:         "updated-tag",
					Color:        sql.NullString{String: "#00ff00", Valid: true},
					Description:  sql.NullString{String: "Updated description", Valid: true},
					DisplayOrder: 0,
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response tagEnvelope
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "1", response.Tag.ID)
				require.Equal(t, "Updated Tag", response.Tag.Name)
			},
		},
		{
			name:           "missing tag ID returns 400",
			tagID:          "",
			requestBody:    updateTagRequest{Name: "Test"},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "tag ID is required", response.Error)
			},
		},
		{
			name:           "invalid tag ID returns 400",
			tagID:          "invalid",
			requestBody:    updateTagRequest{Name: "Test"},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "invalid tag ID", response.Error)
			},
		},
		{
			name:           "invalid body returns 400",
			tagID:          "1",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "invalid request body", response.Error)
			},
		},
		{
			name:  "missing name returns 400",
			tagID: "1",
			requestBody: updateTagRequest{
				Name: "",
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "name is required", response.Error)
			},
		},
		{
			name:  "not found returns 404",
			tagID: "999",
			requestBody: updateTagRequest{
				Name: "Updated Tag",
			},
			setupMock: func(m *mocks.MockStore, _ int64) {
				m.EXPECT().UpdateTag(gomock.Any(), gomock.Any()).Return(db.Tag{}, sql.ErrNoRows)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "tag not found", response.Error)
			},
		},
		{
			name:  "service error returns 500",
			tagID: "1",
			requestBody: updateTagRequest{
				Name: "Updated Tag",
			},
			setupMock: func(m *mocks.MockStore, _ int64) {
				m.EXPECT().
					UpdateTag(gomock.Any(), gomock.Any()).
					Return(db.Tag{}, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "failed to update tag", response.Error)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			handler, mockStore := setupTestHandler(ctrl)
			if tt.setupMock != nil {
				var tagID int64 = 1
				if tt.tagID != "" && tt.tagID != "invalid" {
					parsedID, err := strconv.ParseInt(tt.tagID, 10, 64)
					assert.NoError(t, err, "failed to parse tagID in test setup")
					tagID = parsedID
				}
				tt.setupMock(mockStore, tagID)
			}

			req := createRequest(http.MethodPut, "/tags/"+tt.tagID, tt.requestBody)
			rctx := chi.NewRouteContext()
			if tt.tagID != "" {
				rctx.URLParams.Add("id", tt.tagID)
			}
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()

			handler.handleUpdateTag(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != nil {
				tt.expectedBody(t, w)
			}
		})
	}
}

func TestHandler_handleDeleteTag(t *testing.T) {
	tests := []struct {
		name           string
		tagID          string
		setupMock      func(*mocks.MockStore, int64)
		expectedStatus int
	}{
		{
			name:  "success deletes tag",
			tagID: "1",
			setupMock: func(m *mocks.MockStore, _ int64) {
				m.EXPECT().DeleteTag(gomock.Any(), int64(1)).Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "missing tag ID returns 400",
			tagID:          "",
			setupMock:      func(_ *mocks.MockStore, _ int64) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid tag ID returns 400",
			tagID:          "invalid",
			setupMock:      func(_ *mocks.MockStore, _ int64) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:  "service error returns 500",
			tagID: "1",
			setupMock: func(m *mocks.MockStore, _ int64) {
				m.EXPECT().DeleteTag(gomock.Any(), int64(1)).Return(errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			handler, mockStore := setupTestHandler(ctrl)
			if tt.setupMock != nil {
				var tagID int64 = 1
				if tt.tagID != "" && tt.tagID != "invalid" {
					parsedID, err := strconv.ParseInt(tt.tagID, 10, 64)
					assert.NoError(t, err, "failed to parse tagID in test setup")
					tagID = parsedID
				}
				tt.setupMock(mockStore, tagID)
			}

			req := createRequest(http.MethodDelete, "/tags/"+tt.tagID, nil)
			rctx := chi.NewRouteContext()
			if tt.tagID != "" {
				rctx.URLParams.Add("id", tt.tagID)
			}
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()

			handler.handleDeleteTag(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestHandler_handleReorderTags(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*mocks.MockStore)
		expectedStatus int
		expectedBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "success reorders tags",
			requestBody: reorderTagsRequest{
				TagIDs: []string{"1", "2", "3"},
			},
			setupMock: func(m *mocks.MockStore) {
				// Mock UpdateTagDisplayOrder for each tag
				m.EXPECT().UpdateTagDisplayOrder(gomock.Any(), gomock.Any()).Return(nil).Times(3)
				// Mock ListAllTags to return updated tags
				m.EXPECT().ListAllTags(gomock.Any()).Return([]db.Tag{
					{ID: 1, Name: "Tag 1", Slug: "tag-1", DisplayOrder: 0},
					{ID: 2, Name: "Tag 2", Slug: "tag-2", DisplayOrder: 1},
					{ID: 3, Name: "Tag 3", Slug: "tag-3", DisplayOrder: 2},
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response listTagsResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Len(t, response.Tags, 3)
			},
		},
		{
			name:           "invalid body returns 400",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "empty tagIds returns 400",
			requestBody: reorderTagsRequest{
				TagIDs: []string{},
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Contains(t, response.Error, "tagIds is required")
			},
		},
		{
			name: "invalid tag ID returns 400",
			requestBody: reorderTagsRequest{
				TagIDs: []string{"1", "invalid", "3"},
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Contains(t, response.Error, "invalid tag id")
			},
		},
		{
			name: "service error returns 500",
			requestBody: reorderTagsRequest{
				TagIDs: []string{"1", "2"},
			},
			setupMock: func(m *mocks.MockStore) {
				m.EXPECT().
					UpdateTagDisplayOrder(gomock.Any(), gomock.Any()).
					Return(errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			handler, mockStore := setupTestHandler(ctrl)
			if tt.setupMock != nil {
				tt.setupMock(mockStore)
			}

			req := createRequest(http.MethodPost, "/tags/reorder", tt.requestBody)
			w := httptest.NewRecorder()

			handler.handleReorderTags(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != nil {
				tt.expectedBody(t, w)
			}
		})
	}
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

type errorResponse struct {
	Error string `json:"error"`
}
