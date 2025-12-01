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

// Package api provides the API handling logic for the service.
package api

import (
	"database/sql"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/ajbeattie/octobud/backend/internal/api/notifications"
	"github.com/ajbeattie/octobud/backend/internal/api/repositories"
	"github.com/ajbeattie/octobud/backend/internal/api/rules"
	"github.com/ajbeattie/octobud/backend/internal/api/tags"
	"github.com/ajbeattie/octobud/backend/internal/api/views"
	"github.com/ajbeattie/octobud/backend/internal/core/notification"
	"github.com/ajbeattie/octobud/backend/internal/core/pullrequest"
	"github.com/ajbeattie/octobud/backend/internal/core/repository"
	rulescore "github.com/ajbeattie/octobud/backend/internal/core/rules"
	syncsvc "github.com/ajbeattie/octobud/backend/internal/core/sync"
	"github.com/ajbeattie/octobud/backend/internal/core/tag"
	timelinesvc "github.com/ajbeattie/octobud/backend/internal/core/timeline"
	"github.com/ajbeattie/octobud/backend/internal/core/view"
	"github.com/ajbeattie/octobud/backend/internal/db"
	"github.com/ajbeattie/octobud/backend/internal/github"
	githubinterfaces "github.com/ajbeattie/octobud/backend/internal/github/interfaces"
)

// Handler wires HTTP routes to database-backed operations.
type Handler struct {
	logger         *zap.Logger
	queries        *db.Queries
	notifications  *notification.Service
	syncService    *github.SyncService
	githubClient   githubinterfaces.Client
	timelineSvc    *timelinesvc.Service
	riverClient    db.RiverClient
	notificationsH *notifications.Handler
	tagsH          *tags.Handler
	viewsH         *views.Handler
	rulesH         *rules.Handler
	repositoriesH  *repositories.Handler
}

// HandlerOption configures a Handler
type HandlerOption func(*Handler)

// WithSyncService configures the handler with a sync service for refreshing subject data.
// This enables the refresh-subject endpoint for notifications.
func WithSyncService(dbConn *sql.DB, githubClient githubinterfaces.Client) HandlerOption {
	return func(h *Handler) {
		queries := db.New(dbConn)
		syncSvc := syncsvc.NewService(queries)
		repositorySvc := repository.NewService(queries)
		pullRequestSvc := pullrequest.NewService(queries)
		h.syncService = github.NewSyncService(
			githubClient,
			syncSvc,
			repositorySvc,
			pullRequestSvc,
			queries,
		)
		h.githubClient = githubClient
		h.timelineSvc = timelinesvc.NewService()
	}
}

// WithRiverClient configures the handler with a River client for queuing jobs
func WithRiverClient(client db.RiverClient) HandlerOption {
	return func(h *Handler) {
		h.riverClient = client
	}
}

// NewHandler returns an API handler backed by the provided db queries.
func NewHandler(queries *db.Queries, opts ...HandlerOption) *Handler {
	// Initialize zap logger
	logger, err := zap.NewProduction()
	if err != nil {
		// Fallback to Nop logger if production logger fails
		logger = zap.NewNop()
	}

	// Create all business logic services
	notificationsSvc := notification.NewService(queries)
	repositorySvc := repository.NewService(queries)
	tagSvc := tag.NewService(queries)
	viewSvc := view.NewService(queries)
	ruleSvc := rulescore.NewService(queries)

	h := &Handler{
		logger:        logger,
		queries:       queries,
		notifications: notificationsSvc,
	}

	// Apply options
	for _, opt := range opts {
		opt(h)
	}

	// Create all resource handlers
	h.notificationsH = notifications.New(
		logger, queries, notificationsSvc, repositorySvc, tagSvc,
		h.timelineSvc, h.githubClient, h.syncService, h.riverClient,
	)
	h.tagsH = tags.New(logger, tagSvc)
	h.viewsH = views.New(logger, viewSvc)
	h.rulesH = rules.New(logger, ruleSvc, viewSvc, h.riverClient)
	h.repositoriesH = repositories.New(logger, repositorySvc)

	return h
}

// Register attaches all API routes to the provided router.
// Note: The router passed in should already be scoped to /api (e.g., via router.Route("/api", ...))
func (h *Handler) Register(r chi.Router) {
	h.notificationsH.Register(r)
	h.tagsH.Register(r)
	h.viewsH.Register(r)
	h.rulesH.Register(r)
	h.repositoriesH.Register(r)
}
