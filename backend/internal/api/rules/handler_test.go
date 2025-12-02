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

package rules

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
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/lib/pq"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	rulescore "github.com/ajbeattie/octobud/backend/internal/core/rules"
	"github.com/ajbeattie/octobud/backend/internal/core/view"
	"github.com/ajbeattie/octobud/backend/internal/db"
	"github.com/ajbeattie/octobud/backend/internal/db/mocks"
	"github.com/ajbeattie/octobud/backend/internal/jobs"
	"github.com/ajbeattie/octobud/backend/internal/models"
)

const (
	invalidRuleID = "invalid"
)

func setupTestHandler(
	ctrl *gomock.Controller,
	riverClient db.RiverClient,
) (*Handler, *mocks.MockStore) {
	logger := zap.NewNop()
	mockStore := mocks.NewMockStore(ctrl)
	ruleSvc := rulescore.NewService(mockStore)
	viewSvc := view.NewService(mockStore)
	return New(logger, ruleSvc, viewSvc, riverClient), mockStore
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

func TestHandler_handleListRules(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*mocks.MockStore)
		expectedStatus int
		expectedBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "success returns list of rules",
			setupMock: func(m *mocks.MockStore) {
				actionsJSON, err := json.Marshal(models.RuleActions{SkipInbox: true})
				if err != nil {
					panic("failed to marshal RuleActions in test setup: " + err.Error())
				}
				dbRules := []db.Rule{
					{
						ID:           1,
						Name:         "Test Rule",
						Query:        sql.NullString{String: "is:unread", Valid: true},
						Actions:      actionsJSON,
						Enabled:      true,
						DisplayOrder: 100,
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					},
				}
				m.EXPECT().ListRules(gomock.Any()).Return(dbRules, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response listRulesResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Len(t, response.Rules, 1)
				require.Equal(t, "1", response.Rules[0].ID)
				require.Equal(t, "Test Rule", response.Rules[0].Name)
			},
		},
		{
			name: "service error returns 500",
			setupMock: func(m *mocks.MockStore) {
				m.EXPECT().ListRules(gomock.Any()).Return(nil, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "failed to load rules", response.Error)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			handler, mockStore := setupTestHandler(ctrl, nil)
			if tt.setupMock != nil {
				tt.setupMock(mockStore)
			}

			req := createRequest(http.MethodGet, "/rules", nil)
			w := httptest.NewRecorder()

			handler.handleListRules(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != nil {
				tt.expectedBody(t, w)
			}
		})
	}
}

func TestHandler_handleGetRule(t *testing.T) {
	tests := []struct {
		name           string
		ruleID         string
		setupMock      func(*mocks.MockStore, int64)
		expectedStatus int
		expectedBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:   "success returns rule",
			ruleID: "1",
			setupMock: func(m *mocks.MockStore, _ int64) {
				actionsJSON, err := json.Marshal(models.RuleActions{SkipInbox: true})
				if err != nil {
					panic("failed to marshal RuleActions in test setup: " + err.Error())
				}
				dbRule := db.Rule{
					ID:           1,
					Name:         "Test Rule",
					Query:        sql.NullString{String: "is:unread", Valid: true},
					Actions:      actionsJSON,
					Enabled:      true,
					DisplayOrder: 100,
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				}
				m.EXPECT().GetRule(gomock.Any(), int64(1)).Return(dbRule, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response ruleEnvelope
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "1", response.Rule.ID)
				require.Equal(t, "Test Rule", response.Rule.Name)
			},
		},
		{
			name:           "invalid ID returns 400",
			ruleID:         "invalid",
			setupMock:      func(_ *mocks.MockStore, _ int64) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Contains(t, response.Error, "invalid rule id")
			},
		},
		{
			name:           "missing ID returns 400",
			ruleID:         "",
			setupMock:      func(_ *mocks.MockStore, _ int64) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Contains(t, response.Error, "id is required")
			},
		},
		{
			name:           "zero ID returns 400",
			ruleID:         "0",
			setupMock:      func(_ *mocks.MockStore, _ int64) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Contains(t, response.Error, "invalid rule id")
			},
		},
		{
			name:           "negative ID returns 400",
			ruleID:         "-1",
			setupMock:      func(_ *mocks.MockStore, _ int64) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Contains(t, response.Error, "invalid rule id")
			},
		},
		{
			name:   "not found returns 404",
			ruleID: "999",
			setupMock: func(m *mocks.MockStore, _ int64) {
				m.EXPECT().GetRule(gomock.Any(), int64(999)).Return(db.Rule{}, sql.ErrNoRows)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "rule not found", response.Error)
			},
		},
		{
			name:   "service error returns 500",
			ruleID: "1",
			setupMock: func(m *mocks.MockStore, _ int64) {
				m.EXPECT().
					GetRule(gomock.Any(), int64(1)).
					Return(db.Rule{}, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "failed to get rule", response.Error)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			handler, mockStore := setupTestHandler(ctrl, nil)
			if tt.setupMock != nil {
				var ruleID int64 = 1
				if tt.ruleID != "" && tt.ruleID != invalidRuleID && tt.ruleID != "0" &&
					tt.ruleID != "-1" {
					parsedID, err := strconv.ParseInt(tt.ruleID, 10, 64)
					assert.NoError(t, err, "failed to parse ruleID in test setup")
					ruleID = parsedID
				}
				tt.setupMock(mockStore, ruleID)
			}

			req := createRequest(http.MethodGet, "/rules/"+tt.ruleID, nil)
			rctx := chi.NewRouteContext()
			if tt.ruleID != "" {
				rctx.URLParams.Add("id", tt.ruleID)
			}
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()

			handler.handleGetRule(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != nil {
				tt.expectedBody(t, w)
			}
		})
	}
}

func TestHandler_handleCreateRule(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*mocks.MockStore, *mocks.MockRiverClient)
		expectedStatus int
		expectedBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "success creates rule",
			requestBody: createRuleRequest{
				Name:    "Test Rule",
				Query:   "is:unread",
				Actions: RuleActions{SkipInbox: true},
			},
			setupMock: func(m *mocks.MockStore, _ *mocks.MockRiverClient) {
				// Mock ListRules (for display order calculation)
				m.EXPECT().ListRules(gomock.Any()).Return([]db.Rule{}, nil)
				// Mock CreateRule
				actionsJSON, err := json.Marshal(models.RuleActions{SkipInbox: true})
				if err != nil {
					panic("failed to marshal RuleActions in test setup: " + err.Error())
				}
				m.EXPECT().CreateRule(gomock.Any(), gomock.Any()).Return(db.Rule{
					ID:           1,
					Name:         "Test Rule",
					Query:        sql.NullString{String: "is:unread", Valid: true},
					Actions:      actionsJSON,
					Enabled:      true,
					DisplayOrder: 100,
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				}, nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response ruleEnvelope
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "1", response.Rule.ID)
				require.Equal(t, "Test Rule", response.Rule.Name)
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
			name: "name required error returns 400",
			requestBody: createRuleRequest{
				Name:  "",
				Query: "is:unread",
			},
			setupMock: func(_ *mocks.MockStore, _ *mocks.MockRiverClient) {
				// No mock needed - validation happens before service call
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
			requestBody: createRuleRequest{
				Name:    "Test Rule",
				Query:   "is:unread",
				Actions: RuleActions{},
			},
			setupMock: func(m *mocks.MockStore, _ *mocks.MockRiverClient) {
				m.EXPECT().ListRules(gomock.Any()).Return([]db.Rule{}, nil)
				// Return a PostgreSQL unique violation error
				// The service uses utils.IsUniqueViolation to detect this
				uniqueErr := &pq.Error{Code: "23505"} // unique_violation
				m.EXPECT().CreateRule(gomock.Any(), gomock.Any()).Return(db.Rule{}, uniqueErr)
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
			name: "applyToExisting with riverClient queues job",
			requestBody: createRuleRequest{
				Name:            "Test Rule",
				Query:           "is:unread",
				Actions:         RuleActions{},
				ApplyToExisting: true,
			},
			setupMock: func(m *mocks.MockStore, rc *mocks.MockRiverClient) {
				m.EXPECT().ListRules(gomock.Any()).Return([]db.Rule{}, nil)
				actionsJSON, err := json.Marshal(models.RuleActions{})
				if err != nil {
					panic("failed to marshal RuleActions in test setup: " + err.Error())
				}
				m.EXPECT().CreateRule(gomock.Any(), gomock.Any()).Return(db.Rule{
					ID:           1,
					Name:         "Test Rule",
					Query:        sql.NullString{String: "is:unread", Valid: true},
					Actions:      actionsJSON,
					Enabled:      true,
					DisplayOrder: 100,
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				}, nil)

				rc.EXPECT().
					Insert(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, args river.JobArgs, _ *river.InsertOpts) (*rivertype.JobInsertResult, error) {
						_, ok := args.(jobs.ApplyRuleArgs)
						require.True(t, ok, "expected ApplyRuleArgs")
						return &rivertype.JobInsertResult{}, nil
					})
			},
			expectedStatus: http.StatusCreated,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response ruleEnvelope
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "1", response.Rule.ID)
			},
		},
		{
			name: "service error returns 500",
			requestBody: createRuleRequest{
				Name:    "Test Rule",
				Query:   "is:unread",
				Actions: RuleActions{},
			},
			setupMock: func(m *mocks.MockStore, _ *mocks.MockRiverClient) {
				m.EXPECT().ListRules(gomock.Any()).Return(nil, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "failed to create rule", response.Error)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			riverClient := mocks.NewMockRiverClient(ctrl)
			handler, mockStore := setupTestHandler(ctrl, riverClient)
			if tt.setupMock != nil {
				tt.setupMock(mockStore, riverClient)
			}

			req := createRequest(http.MethodPost, "/rules", tt.requestBody)
			w := httptest.NewRecorder()

			handler.handleCreateRule(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != nil {
				tt.expectedBody(t, w)
			}
		})
	}
}

func TestHandler_handleUpdateRule(t *testing.T) {
	tests := []struct {
		name           string
		ruleID         string
		requestBody    interface{}
		setupMock      func(*mocks.MockStore, int64)
		expectedStatus int
		expectedBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:   "success updates rule",
			ruleID: "1",
			requestBody: updateRuleRequest{
				Name: stringPtr("Updated Rule"),
			},
			setupMock: func(m *mocks.MockStore, _ int64) {
				actionsJSON, err := json.Marshal(models.RuleActions{})
				if err != nil {
					panic("failed to marshal RuleActions in test setup: " + err.Error())
				}
				m.EXPECT().UpdateRule(gomock.Any(), gomock.Any()).Return(db.Rule{
					ID:           1,
					Name:         "Updated Rule",
					Query:        sql.NullString{String: "is:unread", Valid: true},
					Actions:      actionsJSON,
					Enabled:      true,
					DisplayOrder: 100,
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response ruleEnvelope
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "1", response.Rule.ID)
				require.Equal(t, "Updated Rule", response.Rule.Name)
			},
		},
		{
			name:           "invalid ID returns 400",
			ruleID:         "invalid",
			requestBody:    updateRuleRequest{},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid body returns 400",
			ruleID:         "1",
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
			name:   "not found returns 404",
			ruleID: "999",
			requestBody: updateRuleRequest{
				Name: stringPtr("Updated Rule"),
			},
			setupMock: func(m *mocks.MockStore, _ int64) {
				m.EXPECT().UpdateRule(gomock.Any(), gomock.Any()).Return(db.Rule{}, sql.ErrNoRows)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:   "already exists returns 409",
			ruleID: "1",
			requestBody: updateRuleRequest{
				Name: stringPtr("Existing Rule"),
			},
			setupMock: func(m *mocks.MockStore, _ int64) {
				// Return a PostgreSQL unique violation error
				// The service uses utils.IsUniqueViolation to detect this
				uniqueErr := &pq.Error{Code: "23505"} // unique_violation
				m.EXPECT().UpdateRule(gomock.Any(), gomock.Any()).Return(db.Rule{}, uniqueErr)
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name:   "service error returns 500",
			ruleID: "1",
			requestBody: updateRuleRequest{
				Name: stringPtr("Updated Rule"),
			},
			setupMock: func(m *mocks.MockStore, _ int64) {
				m.EXPECT().
					UpdateRule(gomock.Any(), gomock.Any()).
					Return(db.Rule{}, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			handler, mockStore := setupTestHandler(ctrl, nil)
			if tt.setupMock != nil {
				var ruleID int64 = 1
				if tt.ruleID != "" && tt.ruleID != invalidRuleID {
					parsedID, err := strconv.ParseInt(tt.ruleID, 10, 64)
					assert.NoError(t, err, "failed to parse ruleID in test setup")
					ruleID = parsedID
				}
				tt.setupMock(mockStore, ruleID)
			}

			req := createRequest(http.MethodPut, "/rules/"+tt.ruleID, tt.requestBody)
			rctx := chi.NewRouteContext()
			if tt.ruleID != "" {
				rctx.URLParams.Add("id", tt.ruleID)
			}
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()

			handler.handleUpdateRule(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != nil {
				tt.expectedBody(t, w)
			}
		})
	}
}

func TestHandler_handleDeleteRule(t *testing.T) {
	tests := []struct {
		name           string
		ruleID         string
		setupMock      func(*mocks.MockStore, int64)
		expectedStatus int
	}{
		{
			name:   "success deletes rule",
			ruleID: "1",
			setupMock: func(m *mocks.MockStore, _ int64) {
				m.EXPECT().DeleteRule(gomock.Any(), int64(1)).Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "invalid ID returns 400",
			ruleID:         "invalid",
			setupMock:      func(_ *mocks.MockStore, _ int64) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "not found returns 404",
			ruleID: "999",
			setupMock: func(m *mocks.MockStore, _ int64) {
				m.EXPECT().DeleteRule(gomock.Any(), int64(999)).Return(sql.ErrNoRows)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:   "service error returns 500",
			ruleID: "1",
			setupMock: func(m *mocks.MockStore, _ int64) {
				m.EXPECT().DeleteRule(gomock.Any(), int64(1)).Return(errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			handler, mockStore := setupTestHandler(ctrl, nil)
			if tt.setupMock != nil {
				var ruleID int64 = 1
				if tt.ruleID != "" && tt.ruleID != invalidRuleID {
					parsedID, err := strconv.ParseInt(tt.ruleID, 10, 64)
					assert.NoError(t, err, "failed to parse ruleID in test setup")
					ruleID = parsedID
				}
				tt.setupMock(mockStore, ruleID)
			}

			req := createRequest(http.MethodDelete, "/rules/"+tt.ruleID, nil)
			rctx := chi.NewRouteContext()
			if tt.ruleID != "" {
				rctx.URLParams.Add("id", tt.ruleID)
			}
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()

			handler.handleDeleteRule(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestHandler_handleReorderRules(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*mocks.MockStore)
		expectedStatus int
		expectedBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "success reorders rules",
			requestBody: reorderRulesRequest{
				RuleIDs: []string{"1", "2", "3"},
			},
			setupMock: func(m *mocks.MockStore) {
				// Mock UpdateRuleOrder for each rule
				m.EXPECT().UpdateRuleOrder(gomock.Any(), gomock.Any()).Return(nil).Times(3)
				// Mock ListRules to return updated rules
				actionsJSON, err := json.Marshal(models.RuleActions{})
				if err != nil {
					panic("failed to marshal RuleActions in test setup: " + err.Error())
				}
				m.EXPECT().ListRules(gomock.Any()).Return([]db.Rule{

					{
						ID:           1,
						Name:         "Rule 1",
						Actions:      actionsJSON,
						Enabled:      true,
						DisplayOrder: 100,
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					},

					{
						ID:           2,
						Name:         "Rule 2",
						Actions:      actionsJSON,
						Enabled:      true,
						DisplayOrder: 200,
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					},

					{
						ID:           3,
						Name:         "Rule 3",
						Actions:      actionsJSON,
						Enabled:      true,
						DisplayOrder: 300,
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					},
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response listRulesResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Len(t, response.Rules, 3)
			},
		},
		{
			name:           "invalid body returns 400",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "empty ruleIds returns 400",
			requestBody: reorderRulesRequest{
				RuleIDs: []string{},
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Contains(t, response.Error, "ruleIds cannot be empty")
			},
		},
		{
			name: "invalid rule ID returns 400",
			requestBody: reorderRulesRequest{
				RuleIDs: []string{"1", invalidRuleID, "3"},
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Contains(t, response.Error, "invalid rule id")
			},
		},
		{
			name: "not found returns 404",
			requestBody: reorderRulesRequest{
				RuleIDs: []string{"999"},
			},
			setupMock: func(m *mocks.MockStore) {
				m.EXPECT().UpdateRuleOrder(gomock.Any(), gomock.Any()).Return(sql.ErrNoRows)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "service error returns 500",
			requestBody: reorderRulesRequest{
				RuleIDs: []string{"1", "2"},
			},
			setupMock: func(m *mocks.MockStore) {
				m.EXPECT().
					UpdateRuleOrder(gomock.Any(), gomock.Any()).
					Return(errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			handler, mockStore := setupTestHandler(ctrl, nil)
			if tt.setupMock != nil {
				tt.setupMock(mockStore)
			}

			req := createRequest(http.MethodPost, "/rules/reorder", tt.requestBody)
			w := httptest.NewRecorder()

			handler.handleReorderRules(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != nil {
				tt.expectedBody(t, w)
			}
		})
	}
}

func TestHandler_parseRuleIDParam(t *testing.T) {
	tests := []struct {
		name        string
		ruleID      string
		expectError bool
		expectedID  int64
	}{
		{
			name:        "valid ID",
			ruleID:      "123",
			expectError: false,
			expectedID:  123,
		},
		{
			name:        "missing ID",
			ruleID:      "",
			expectError: true,
		},
		{
			name:        "invalid format",
			ruleID:      "abc",
			expectError: true,
		},
		{
			name:        "zero ID",
			ruleID:      "0",
			expectError: true,
		},
		{
			name:        "negative ID",
			ruleID:      "-1",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/rules/"+tt.ruleID, http.NoBody)
			rctx := chi.NewRouteContext()
			if tt.ruleID != "" {
				rctx.URLParams.Add("id", tt.ruleID)
			}
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			ruleID, err := parseRuleIDParam(req)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedID, ruleID)
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
