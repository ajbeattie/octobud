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

package notifications

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/ajbeattie/octobud/backend/internal/api/shared"
	"github.com/ajbeattie/octobud/backend/internal/models"
)

// Error definitions
var (
	ErrFailedToParseListOptions          = errors.New("failed to parse notification list options")
	ErrFailedToLoadNotifications         = errors.New("failed to load notifications")
	ErrFailedToLoadRepositories          = errors.New("failed to load repositories")
	ErrFailedToBuildNotificationResponse = errors.New("failed to build notification response")
	ErrInvalidGithubIDEncoding           = errors.New("invalid githubID encoding")
	ErrFailedToLoadNotification          = errors.New("failed to load notification")
	ErrSyncServiceNotAvailable           = errors.New("sync service not available")
	ErrFailedToRefreshSubjectData        = errors.New("failed to refresh subject data")
	ErrFailedToLoadUpdatedNotification   = errors.New("failed to load updated notification")
	ErrFailedToFetchNotification         = errors.New("failed to fetch notification")
	ErrFailedToParseSubjectInfo          = errors.New("failed to parse subject info")
	ErrGitHubClientNotConfigured         = errors.New("GitHub client not configured")
	ErrGitHubClientTypeMismatch          = errors.New("GitHub client type mismatch")
	ErrFailedToFetchTimeline             = errors.New("failed to fetch timeline")
	ErrFailedToFetchTags                 = errors.New("failed to fetch tags")
)

func (h *Handler) handleListNotifications(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	options := parseNotificationListOptions(r)

	result, err := h.notifications.ListNotifications(ctx, options)
	if err != nil {
		h.logger.Error(
			"failed to load notifications",
			zap.Error(errors.Join(ErrFailedToLoadNotifications, err)),
		)
		shared.WriteError(w, http.StatusInternalServerError, "failed to load notifications")
		return
	}

	shared.WriteJSON(w, http.StatusOK, listNotificationsResponse{
		Notifications: result.Notifications,
		Total:         result.Total,
		Page:          result.Page,
		PageSize:      result.PageSize,
	})
}

func (h *Handler) handlePollNotifications(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	options := parseNotificationListOptions(r)

	result, err := h.notifications.ListPollNotifications(ctx, options)
	if err != nil {
		h.logger.Error(
			"failed to load poll notifications",
			zap.Error(errors.Join(ErrFailedToLoadNotifications, err)),
		)
		shared.WriteError(w, http.StatusInternalServerError, "failed to load notifications")
		return
	}

	// Convert models.PollNotification to PollNotificationResponse
	notifications := make([]PollNotificationResponse, 0, len(result.Notifications))
	for _, n := range result.Notifications {
		notifications = append(notifications, PollNotificationResponse{
			ID:                n.ID,
			GithubID:          n.GithubID,
			EffectiveSortDate: n.EffectiveSortDate,
			Archived:          n.Archived,
			Muted:             n.Muted,
			RepoFullName:      n.RepoFullName,
			SubjectTitle:      n.SubjectTitle,
			SubjectType:       n.SubjectType,
			Reason:            n.Reason,
		})
	}

	shared.WriteJSON(w, http.StatusOK, listPollNotificationsResponse{
		Notifications: notifications,
		Total:         result.Total,
		Page:          result.Page,
		PageSize:      result.PageSize,
	})
}

func (h *Handler) handleGetNotification(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	rawGithubID := chi.URLParam(r, "githubID")
	if rawGithubID == "" {
		shared.WriteError(w, http.StatusBadRequest, "githubID is required")
		return
	}

	githubID, err := url.PathUnescape(rawGithubID)
	if err != nil {
		h.logger.Error(
			"invalid githubID encoding",
			zap.String("github_id", rawGithubID),
			zap.Error(errors.Join(ErrInvalidGithubIDEncoding, err)),
		)
		shared.WriteError(w, http.StatusBadRequest, "invalid githubID encoding")
		return
	}

	queryStr := r.URL.Query().Get("query")
	notification, err := h.notifications.GetNotificationWithDetails(ctx, githubID, queryStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.logger.Debug("notification not found", zap.String("github_id", githubID))
			shared.WriteError(w, http.StatusNotFound, "notification not found")
			return
		}
		h.logger.Error(
			"failed to load notification",
			zap.String("github_id", githubID),
			zap.Error(errors.Join(ErrFailedToLoadNotification, err)),
		)
		shared.WriteError(w, http.StatusInternalServerError, "failed to load notification")
		return
	}

	shared.WriteJSON(w, http.StatusOK, notificationDetailResponse{Notification: notification})
}

func (h *Handler) handleRefreshNotificationSubject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check if sync service is available
	if h.syncService == nil {
		h.logger.Warn(
			"sync service not available for refresh",
			zap.Error(ErrSyncServiceNotAvailable),
		)
		shared.WriteError(w, http.StatusServiceUnavailable, "subject refresh not available")
		return
	}

	rawGithubID := chi.URLParam(r, "githubID")
	if rawGithubID == "" {
		shared.WriteError(w, http.StatusBadRequest, "githubID is required")
		return
	}

	githubID, err := url.PathUnescape(rawGithubID)
	if err != nil {
		h.logger.Error(
			"invalid githubID encoding",
			zap.String("github_id", rawGithubID),
			zap.Error(errors.Join(ErrInvalidGithubIDEncoding, err)),
		)
		shared.WriteError(w, http.StatusBadRequest, "invalid githubID encoding")
		return
	}

	// Verify the notification exists
	_, err = h.notifications.GetByGithubID(ctx, githubID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.logger.Debug("notification not found for refresh", zap.String("github_id", githubID))
			shared.WriteError(w, http.StatusNotFound, "notification not found")
			return
		}
		h.logger.Error(
			"failed to load notification for refresh",
			zap.String("github_id", githubID),
			zap.Error(errors.Join(ErrFailedToLoadNotification, err)),
		)
		shared.WriteError(w, http.StatusInternalServerError, "failed to load notification")
		return
	}

	// Refresh the subject by fetching fresh data from GitHub
	err = h.refreshSubjectData(ctx, githubID)
	if err != nil {
		h.logger.Error(
			"failed to refresh subject data",
			zap.String("github_id", githubID),
			zap.Error(errors.Join(ErrFailedToRefreshSubjectData, err)),
		)
		shared.WriteError(w, http.StatusInternalServerError, "failed to refresh subject data")
		return
	}

	// Get the updated notification with details
	queryStr := r.URL.Query().Get("query")
	updatedNotification, err := h.notifications.GetNotificationWithDetails(ctx, githubID, queryStr)
	if err != nil {
		h.logger.Error(
			"failed to load updated notification",
			zap.String("github_id", githubID),
			zap.Error(errors.Join(ErrFailedToLoadUpdatedNotification, err)),
		)
		shared.WriteError(w, http.StatusInternalServerError, "failed to load updated notification")
		return
	}

	shared.WriteJSON(
		w,
		http.StatusOK,
		refreshNotificationSubjectResponse{Notification: updatedNotification},
	)
}

func parseNotificationListOptions(r *http.Request) models.ListOptions {
	query := r.URL.Query()

	opts := models.ListOptions{
		Query: query.Get(
			"query",
		), // Combined query string with key-value pairs and free text
		Page:     parseIntDefault(query.Get("page")),
		PageSize: parseIntDefault(query.Get("pageSize")),
		IncludeSubject: parseBoolDefault(
			query.Get("includeSubject"),
		), // Default: false to reduce payload size
	}

	return opts
}

func parseIntDefault(raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return value
}

func parseBoolDefault(raw string) bool {
	raw = strings.TrimSpace(strings.ToLower(raw))
	return raw == "true" || raw == "1" || raw == "yes"
}

// refreshSubjectData fetches fresh subject data from GitHub and updates the notification
func (h *Handler) refreshSubjectData(ctx context.Context, githubID string) error {
	if h.syncService == nil {
		return ErrSyncServiceNotAvailable
	}

	err := h.syncService.RefreshSubjectData(ctx, githubID)
	if err != nil {
		return errors.Join(ErrFailedToRefreshSubjectData, err)
	}
	return nil
}
