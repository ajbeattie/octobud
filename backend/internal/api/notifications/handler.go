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
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/ajbeattie/octobud/backend/internal/core/notification"
	"github.com/ajbeattie/octobud/backend/internal/core/repository"
	"github.com/ajbeattie/octobud/backend/internal/core/tag"
	timelinesvc "github.com/ajbeattie/octobud/backend/internal/core/timeline"
	"github.com/ajbeattie/octobud/backend/internal/db"
	githubinterfaces "github.com/ajbeattie/octobud/backend/internal/github/interfaces"
	"github.com/ajbeattie/octobud/backend/internal/sync"
)

// Handler handles notification-related HTTP routes
type Handler struct {
	logger        *zap.Logger
	queries       *db.Queries
	notifications notification.NotificationService
	repositorySvc repository.RepositoryService
	tagSvc        tag.TagService
	timelineSvc   *timelinesvc.Service
	githubClient  githubinterfaces.Client
	syncService   *sync.Service
	riverClient   db.RiverClient
}

// New creates a new notifications handler
func New(
	logger *zap.Logger,
	queries *db.Queries,
	notificationsSvc notification.NotificationService,
	repositorySvc repository.RepositoryService,
	tagSvc tag.TagService,
	timelineSvc *timelinesvc.Service,
	githubClient githubinterfaces.Client,
	syncService *sync.Service,
	riverClient db.RiverClient,
) *Handler {
	return &Handler{
		logger:        logger,
		queries:       queries,
		notifications: notificationsSvc,
		repositorySvc: repositorySvc,
		tagSvc:        tagSvc,
		timelineSvc:   timelineSvc,
		githubClient:  githubClient,
		syncService:   syncService,
		riverClient:   riverClient,
	}
}

// Register registers notification routes on the provided router
func (h *Handler) Register(r chi.Router) {
	r.Route("/notifications", func(r chi.Router) {
		r.Get("/", h.handleListNotifications)
		r.Get("/poll", h.handlePollNotifications) // Poll endpoint for service worker polling
		r.Get("/{githubID}", h.handleGetNotification)
		r.Get("/{githubID}/timeline", h.handleGetNotificationTimeline)
		r.Post("/{githubID}/refresh-subject", h.handleRefreshNotificationSubject)

		// Bulk operations - MUST come before individual routes to avoid "bulk" being treated as a githubID
		r.Post("/bulk/mark-read", h.handleBulkMarkNotificationsRead)
		r.Post("/bulk/mark-unread", h.handleBulkMarkNotificationsUnread)
		r.Post("/bulk/archive", h.handleBulkArchiveNotifications)
		r.Post("/bulk/unarchive", h.handleBulkUnarchiveNotifications)
		r.Post("/bulk/mute", h.handleBulkMuteNotifications)
		r.Post("/bulk/unmute", h.handleBulkUnmuteNotifications)
		r.Post("/bulk/snooze", h.handleBulkSnoozeNotifications)
		r.Post("/bulk/unsnooze", h.handleBulkUnsnoozeNotifications) // Uses unified handler
		r.Post("/bulk/star", h.handleBulkStarNotifications)
		r.Post("/bulk/unstar", h.handleBulkUnstarNotifications)
		r.Post("/bulk/unfilter", h.handleBulkUnfilterNotifications)
		r.Post("/bulk/assign-tag", h.handleBulkAssignTag)
		r.Post("/bulk/remove-tag", h.handleBulkRemoveTag)

		// Action-based endpoints
		r.Post("/{githubID}/mark-read", h.handleMarkNotificationRead)
		r.Post("/{githubID}/mark-unread", h.handleMarkNotificationUnread)
		r.Post("/{githubID}/archive", h.handleArchiveNotification)
		r.Post("/{githubID}/unarchive", h.handleUnarchiveNotification)
		r.Post("/{githubID}/mute", h.handleMuteNotification)
		r.Post("/{githubID}/unmute", h.handleUnmuteNotification)
		r.Post("/{githubID}/snooze", h.handleSnoozeNotification)
		r.Post("/{githubID}/unsnooze", h.handleUnsnoozeNotification)
		r.Post("/{githubID}/star", h.handleStarNotification)
		r.Post("/{githubID}/unstar", h.handleUnstarNotification)
		r.Post("/{githubID}/unfilter", h.handleUnfilterNotification)

		// Tag operations
		r.Post("/{githubID}/tags", h.handleAssignTagToNotification)
		r.Post("/{githubID}/tags-by-name", h.handleAssignTagByName)
		r.Delete("/{githubID}/tags/{tagId}", h.handleRemoveTagFromNotification)
	})
}
