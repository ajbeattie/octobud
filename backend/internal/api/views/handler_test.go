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

package views

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	viewcore "github.com/ajbeattie/octobud/backend/internal/core/view"
	viewmocks "github.com/ajbeattie/octobud/backend/internal/core/view/mocks"
	"github.com/ajbeattie/octobud/backend/internal/models"
)

func setupTestHandler(ctrl *gomock.Controller) (*Handler, *viewmocks.MockViewService) {
	logger := zap.NewNop()
	mockViewSvc := viewmocks.NewMockViewService(ctrl)
	handler := &Handler{
		logger:  logger,
		viewSvc: mockViewSvc,
	}
	return handler, mockViewSvc
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

func TestHandler_handleListViews(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*viewmocks.MockViewService)
		expectedStatus int
		expectedBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "success returns list of views",
			setupMock: func(mockSvc *viewmocks.MockViewService) {
				mockSvc.EXPECT().
					ListViewsWithCounts(gomock.Any()).
					Return([]models.View{
						{
							ID:           "1",
							Name:         "Test View",
							Slug:         "test-view",
							Query:        "is:unread",
							IsDefault:    false,
							DisplayOrder: 100,
							UnreadCount:  5,
						},
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response listViewsResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.GreaterOrEqual(t, len(response.Views), 1)
				require.Equal(t, "1", response.Views[0].ID)
				require.Equal(t, "Test View", response.Views[0].Name)
			},
		},
		{
			name: "service error returns 500",
			setupMock: func(mockSvc *viewmocks.MockViewService) {
				mockSvc.EXPECT().
					ListViewsWithCounts(gomock.Any()).
					Return(nil, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "failed to load views", response.Error)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			handler, mockSvc := setupTestHandler(ctrl)
			if tt.setupMock != nil {
				tt.setupMock(mockSvc)
			}

			req := createRequest(http.MethodGet, "/views", nil)
			w := httptest.NewRecorder()

			handler.handleListViews(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != nil {
				tt.expectedBody(t, w)
			}
		})
	}
}

func TestHandler_handleCreateView(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*viewmocks.MockViewService)
		expectedStatus int
		expectedBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "success creates view",
			requestBody: createViewRequest{
				Name:        "Test View",
				Description: stringPtr("Test description"),
				Icon:        stringPtr("test-icon"),
				Query:       "is:unread",
			},
			setupMock: func(mockSvc *viewmocks.MockViewService) {
				mockSvc.EXPECT().
					CreateView(gomock.Any(), "Test View", stringPtr("Test description"), stringPtr("test-icon"), gomock.Any(), "is:unread").
					Return(models.View{
						ID:           "1",
						Name:         "Test View",
						Slug:         "test-view",
						Description:  stringPtr("Test description"),
						Icon:         stringPtr("test-icon"),
						Query:        "is:unread",
						IsDefault:    false,
						DisplayOrder: 100,
						UnreadCount:  0,
					}, nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response viewEnvelope
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "1", response.View.ID)
				require.Equal(t, "Test View", response.View.Name)
			},
		},
		{
			name:           "invalid body returns 400",
			requestBody:    "invalid json",
			setupMock:      func(*viewmocks.MockViewService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "invalid request body", response.Error)
			},
		},
		{
			name: "name required error returns 400",
			requestBody: createViewRequest{
				Name:  "",
				Query: "is:unread",
			},
			setupMock: func(mockSvc *viewmocks.MockViewService) {
				mockSvc.EXPECT().
					CreateView(gomock.Any(), "", gomock.Any(), gomock.Any(), gomock.Any(), "is:unread").
					Return(models.View{}, viewcore.ErrNameRequired)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Contains(t, response.Error, "name is required")
			},
		},
		{
			name: "already exists error returns 409",
			requestBody: createViewRequest{
				Name:  "Test View",
				Query: "is:unread",
			},
			setupMock: func(mockSvc *viewmocks.MockViewService) {
				mockSvc.EXPECT().
					CreateView(gomock.Any(), "Test View", gomock.Any(), gomock.Any(), gomock.Any(), "is:unread").
					Return(models.View{}, viewcore.ErrViewNameAlreadyExists)
			},
			expectedStatus: http.StatusConflict,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Contains(t, response.Error, "already exists")
			},
		},
		{
			name: "unique violation error (fallback) returns 409",
			requestBody: createViewRequest{
				Name:  "Test View",
				Query: "is:unread",
			},
			setupMock: func(mockSvc *viewmocks.MockViewService) {
				// Return a raw PostgreSQL unique violation error (not wrapped)
				// This tests the fallback path in the handler
				uniqueErr := &pq.Error{Code: "23505"} // unique_violation
				mockSvc.EXPECT().
					CreateView(gomock.Any(), "Test View", gomock.Any(), gomock.Any(), gomock.Any(), "is:unread").
					Return(models.View{}, uniqueErr)
			},
			expectedStatus: http.StatusConflict,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "a view with that name already exists", response.Error)
			},
		},
		{
			name: "invalid query error returns 400",
			requestBody: createViewRequest{
				Name:  "Test View",
				Query: "invalid:query:format",
			},
			setupMock: func(mockSvc *viewmocks.MockViewService) {
				mockSvc.EXPECT().
					CreateView(gomock.Any(), "Test View", gomock.Any(), gomock.Any(), gomock.Any(), "invalid:query:format").
					Return(models.View{}, viewcore.ErrInvalidQuery)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Contains(t, response.Error, "invalid")
			},
		},
		{
			name: "service error returns 500",
			requestBody: createViewRequest{
				Name:  "Test View",
				Query: "is:unread",
			},
			setupMock: func(mockSvc *viewmocks.MockViewService) {
				mockSvc.EXPECT().
					CreateView(gomock.Any(), "Test View", gomock.Any(), gomock.Any(), gomock.Any(), "is:unread").
					Return(models.View{}, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "failed to create view", response.Error)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			handler, mockSvc := setupTestHandler(ctrl)
			if tt.setupMock != nil {
				tt.setupMock(mockSvc)
			}

			req := createRequest(http.MethodPost, "/views", tt.requestBody)
			w := httptest.NewRecorder()

			handler.handleCreateView(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != nil {
				tt.expectedBody(t, w)
			}
		})
	}
}

func TestHandler_handleUpdateView(t *testing.T) {
	tests := []struct {
		name           string
		viewID         string
		requestBody    interface{}
		setupMock      func(*viewmocks.MockViewService, int64)
		expectedStatus int
		expectedBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:   "success updates view",
			viewID: "1",
			requestBody: updateViewRequest{
				Name: stringPtr("Updated View"),
			},
			setupMock: func(mockSvc *viewmocks.MockViewService, _ int64) {
				mockSvc.EXPECT().
					UpdateView(gomock.Any(), int64(1), stringPtr("Updated View"), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(models.View{
						ID:           "1",
						Name:         "Updated View",
						Slug:         "updated-view",
						Query:        "is:unread",
						IsDefault:    false,
						DisplayOrder: 100,
						UnreadCount:  0,
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response viewEnvelope
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "1", response.View.ID)
				require.Equal(t, "Updated View", response.View.Name)
			},
		},
		{
			name:           "invalid ID returns 400",
			viewID:         "invalid",
			requestBody:    updateViewRequest{},
			setupMock:      func(*viewmocks.MockViewService, int64) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid body returns 400",
			viewID:         "1",
			requestBody:    "invalid json",
			setupMock:      func(*viewmocks.MockViewService, int64) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "invalid request body", response.Error)
			},
		},
		{
			name:   "not found returns 404",
			viewID: "999",
			requestBody: updateViewRequest{
				Name: stringPtr("Updated View"),
			},
			setupMock: func(mockSvc *viewmocks.MockViewService, _ int64) {
				mockSvc.EXPECT().
					UpdateView(gomock.Any(), int64(999), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(models.View{}, viewcore.ErrViewNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:   "already exists returns 409",
			viewID: "1",
			requestBody: updateViewRequest{
				Name: stringPtr("Existing View"),
			},
			setupMock: func(mockSvc *viewmocks.MockViewService, _ int64) {
				mockSvc.EXPECT().
					UpdateView(gomock.Any(), int64(1), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(models.View{}, viewcore.ErrViewNameAlreadyExists)
			},
			expectedStatus: http.StatusConflict,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Contains(t, response.Error, "already exists")
			},
		},
		{
			name:   "unique violation error (fallback) returns 409",
			viewID: "1",
			requestBody: updateViewRequest{
				Name: stringPtr("Existing View"),
			},
			setupMock: func(mockSvc *viewmocks.MockViewService, _ int64) {
				// Return a raw PostgreSQL unique violation error (not wrapped)
				// This tests the fallback path in the handler
				uniqueErr := &pq.Error{Code: "23505"} // unique_violation
				mockSvc.EXPECT().
					UpdateView(gomock.Any(), int64(1), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(models.View{}, uniqueErr)
			},
			expectedStatus: http.StatusConflict,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "a view with that name already exists", response.Error)
			},
		},
		{
			name:   "invalid query error returns 400",
			viewID: "1",
			requestBody: updateViewRequest{
				Query: stringPtr("invalid:query:format"),
			},
			setupMock: func(mockSvc *viewmocks.MockViewService, _ int64) {
				mockSvc.EXPECT().
					UpdateView(gomock.Any(), int64(1), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), stringPtr("invalid:query:format")).
					Return(models.View{}, viewcore.ErrInvalidQuery)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "service error returns 500",
			viewID: "1",
			requestBody: updateViewRequest{
				Name: stringPtr("Updated View"),
			},
			setupMock: func(mockSvc *viewmocks.MockViewService, _ int64) {
				mockSvc.EXPECT().
					UpdateView(gomock.Any(), int64(1), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(models.View{}, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			var viewID int64 = 1
			if tt.viewID != "" && tt.viewID != "invalid" {
				parsedID, err := strconv.ParseInt(tt.viewID, 10, 64)
				assert.NoError(t, err, "failed to parse viewID in test setup")
				viewID = parsedID
			}
			handler, mockSvc := setupTestHandler(ctrl)
			if tt.setupMock != nil {
				tt.setupMock(mockSvc, viewID)
			}

			req := createRequest(http.MethodPut, "/views/"+tt.viewID, tt.requestBody)
			rctx := chi.NewRouteContext()
			if tt.viewID != "" {
				rctx.URLParams.Add("id", tt.viewID)
			}
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()

			handler.handleUpdateView(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != nil {
				tt.expectedBody(t, w)
			}
		})
	}
}

func TestHandler_handleDeleteView(t *testing.T) {
	tests := []struct {
		name           string
		viewID         string
		force          bool
		setupMock      func(*viewmocks.MockViewService, int64)
		expectedStatus int
		expectedBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:   "success deletes view",
			viewID: "1",
			force:  false,
			setupMock: func(mockSvc *viewmocks.MockViewService, _ int64) {
				mockSvc.EXPECT().
					DeleteView(gomock.Any(), int64(1), false).
					Return(0, nil) // No linked rules
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "invalid ID returns 400",
			viewID:         "invalid",
			force:          false,
			setupMock:      func(*viewmocks.MockViewService, int64) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "not found returns 404",
			viewID: "999",
			force:  false,
			setupMock: func(mockSvc *viewmocks.MockViewService, _ int64) {
				mockSvc.EXPECT().
					DeleteView(gomock.Any(), int64(999), false).
					Return(0, viewcore.ErrViewNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:   "linked rules conflict returns 409",
			viewID: "1",
			force:  false,
			setupMock: func(mockSvc *viewmocks.MockViewService, _ int64) {
				mockSvc.EXPECT().
					DeleteView(gomock.Any(), int64(1), false).
					Return(2, nil) // 2 linked rules
			},
			expectedStatus: http.StatusConflict,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Contains(t, response["error"], "linked rules")
				require.Equal(t, float64(2), response["linkedRuleCount"])
			},
		},
		{
			name:   "service error returns 500",
			viewID: "1",
			force:  false,
			setupMock: func(mockSvc *viewmocks.MockViewService, _ int64) {
				mockSvc.EXPECT().
					DeleteView(gomock.Any(), int64(1), false).
					Return(0, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:   "force delete with linked rules succeeds",
			viewID: "1",
			force:  true,
			setupMock: func(mockSvc *viewmocks.MockViewService, _ int64) {
				mockSvc.EXPECT().
					DeleteView(gomock.Any(), int64(1), true).
					Return(2, nil) // 2 linked rules, but force=true so deletion proceeds
			},
			expectedStatus: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			var viewID int64 = 1
			if tt.viewID != "" && tt.viewID != "invalid" {
				parsedID, err := strconv.ParseInt(tt.viewID, 10, 64)
				assert.NoError(t, err, "failed to parse viewID in test setup")
				viewID = parsedID
			}
			handler, mockSvc := setupTestHandler(ctrl)
			if tt.setupMock != nil {
				tt.setupMock(mockSvc, viewID)
			}

			url := "/views/" + tt.viewID
			if tt.force {
				url += "?force=true"
			}
			req := createRequest(http.MethodDelete, url, nil)
			rctx := chi.NewRouteContext()
			if tt.viewID != "" {
				rctx.URLParams.Add("id", tt.viewID)
			}
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()

			handler.handleDeleteView(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != nil {
				tt.expectedBody(t, w)
			}
		})
	}
}

func TestHandler_handleReorderViews(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*viewmocks.MockViewService)
		expectedStatus int
		expectedBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "success reorders views",
			requestBody: reorderViewsRequest{
				ViewIDs: []string{"1", "2", "3"},
			},
			setupMock: func(mockSvc *viewmocks.MockViewService) {
				mockSvc.EXPECT().
					ReorderViews(gomock.Any(), []int64{1, 2, 3}).
					Return([]models.View{

						{
							ID:           "1",
							Name:         "View 1",
							Slug:         "view-1",
							Query:        "is:unread",
							IsDefault:    false,
							DisplayOrder: 100,
							UnreadCount:  0,
						},
						{
							ID:           "2",
							Name:         "View 2",
							Slug:         "view-2",
							Query:        "is:read",
							IsDefault:    false,
							DisplayOrder: 200,
							UnreadCount:  0,
						},

						{
							ID:           "3",
							Name:         "View 3",
							Slug:         "view-3",
							Query:        "is:starred",
							IsDefault:    false,
							DisplayOrder: 300,
							UnreadCount:  0,
						},
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response listViewsResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Len(t, response.Views, 3)
			},
		},
		{
			name:           "invalid body returns 400",
			requestBody:    "invalid json",
			setupMock:      func(*viewmocks.MockViewService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "empty viewIds returns 400",
			requestBody: reorderViewsRequest{
				ViewIDs: []string{},
			},
			setupMock:      func(*viewmocks.MockViewService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Contains(t, response.Error, "viewIds cannot be empty")
			},
		},
		{
			name: "invalid view ID returns 400",
			requestBody: reorderViewsRequest{
				ViewIDs: []string{"1", "invalid", "3"},
			},
			setupMock:      func(*viewmocks.MockViewService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Contains(t, response.Error, "invalid view id")
			},
		},
		{
			name: "not found returns 404",
			requestBody: reorderViewsRequest{
				ViewIDs: []string{"999"},
			},
			setupMock: func(mockSvc *viewmocks.MockViewService) {
				mockSvc.EXPECT().
					ReorderViews(gomock.Any(), []int64{999}).
					Return(nil, viewcore.ErrViewNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "system view error returns 400",
			requestBody: reorderViewsRequest{
				ViewIDs: []string{"1"},
			},
			setupMock: func(mockSvc *viewmocks.MockViewService) {
				mockSvc.EXPECT().
					ReorderViews(gomock.Any(), []int64{1}).
					Return(nil, viewcore.ErrCannotReorderSystemView)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Contains(t, response.Error, "system view")
			},
		},
		{
			name: "service error returns 500",
			requestBody: reorderViewsRequest{
				ViewIDs: []string{"1", "2"},
			},
			setupMock: func(mockSvc *viewmocks.MockViewService) {
				mockSvc.EXPECT().
					ReorderViews(gomock.Any(), []int64{1, 2}).
					Return(nil, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			handler, mockSvc := setupTestHandler(ctrl)
			if tt.setupMock != nil {
				tt.setupMock(mockSvc)
			}

			req := createRequest(http.MethodPost, "/views/reorder", tt.requestBody)
			w := httptest.NewRecorder()

			handler.handleReorderViews(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != nil {
				tt.expectedBody(t, w)
			}
		})
	}
}

func TestHandler_parseViewIDParam(t *testing.T) {
	tests := []struct {
		name        string
		viewID      string
		expectError bool
		expectedID  int64
	}{
		{
			name:        "valid ID",
			viewID:      "123",
			expectError: false,
			expectedID:  123,
		},
		{
			name:        "missing ID",
			viewID:      "",
			expectError: true,
		},
		{
			name:        "invalid format",
			viewID:      "abc",
			expectError: true,
		},
		{
			name:        "zero ID",
			viewID:      "0",
			expectError: true,
		},
		{
			name:        "negative ID",
			viewID:      "-1",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/views/"+tt.viewID, http.NoBody)
			rctx := chi.NewRouteContext()
			if tt.viewID != "" {
				rctx.URLParams.Add("id", tt.viewID)
			}
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			viewID, err := parseViewIDParam(req)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedID, viewID)
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
