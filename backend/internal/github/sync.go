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

// Package github provides the GitHub sync service.
package github

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/sqlc-dev/pqtype"
	"go.uber.org/zap"

	"github.com/ajbeattie/octobud/backend/internal/core/pullrequest"
	"github.com/ajbeattie/octobud/backend/internal/core/repository"
	"github.com/ajbeattie/octobud/backend/internal/core/sync"
	"github.com/ajbeattie/octobud/backend/internal/db"
	githubinterfaces "github.com/ajbeattie/octobud/backend/internal/github/interfaces"
	"github.com/ajbeattie/octobud/backend/internal/github/types"
	"github.com/ajbeattie/octobud/backend/internal/models"
)

// Error definitions
var (
	ErrFailedToGetSyncState              = errors.New("failed to get sync state")
	ErrFailedToUpdateSyncState           = errors.New("failed to update sync state")
	ErrFailedToFetchNotifications        = errors.New("failed to fetch notifications")
	ErrFailedToUpsertRepository          = errors.New("failed to upsert repository")
	ErrFailedToUpsertNotification        = errors.New("failed to upsert notification")
	ErrFailedToGetNotification           = errors.New("failed to get notification")
	ErrNotificationMissingSubjectURL     = errors.New("notification has no subject URL")
	ErrFailedToFetchSubject              = errors.New("failed to fetch subject")
	ErrFailedToGetRepository             = errors.New("failed to get repository")
	ErrFailedToUpdateNotificationSubject = errors.New("failed to update notification subject")
	ErrFailedToExtractPullRequestData    = errors.New("failed to extract pull request data")
	ErrFailedToUpsertPullRequest         = errors.New("failed to upsert pull request")
)

// SyncService coordinates fetching notifications from GitHub and persisting them.
type SyncService struct {
	client             githubinterfaces.Client
	clock              func() time.Time
	logger             *zap.Logger
	syncService        *sync.Service
	repositoryService  *repository.Service
	pullRequestService *pullrequest.Service
	queries            db.Store // Used for notification operations (to avoid import cycle with notification service)
}

// SyncOption mutates the SyncService configuration.
type SyncOption func(*SyncService)

// WithClock injects a clock function, primarily for testing.
func WithClock(clock func() time.Time) SyncOption {
	return func(s *SyncService) {
		if clock != nil {
			s.clock = clock
		}
	}
}

// WithLogger injects a zap logger.
func WithLogger(logger *zap.Logger) SyncOption {
	return func(s *SyncService) {
		if logger != nil {
			s.logger = logger
		}
	}
}

// NewSyncService assembles a SyncService with the provided dependencies.
func NewSyncService(
	client githubinterfaces.Client,
	syncService *sync.Service,
	repositoryService *repository.Service,
	pullRequestService *pullrequest.Service,
	queries db.Store,
	opts ...SyncOption,
) *SyncService {
	service := &SyncService{
		client:             client,
		clock:              time.Now,
		logger:             zap.NewNop(), // Default to no-op logger
		syncService:        syncService,
		repositoryService:  repositoryService,
		pullRequestService: pullRequestService,
		queries:            queries,
	}

	for _, opt := range opts {
		opt(service)
	}

	return service
}

// GitHubClient returns the GitHub client used by this service.
func (s *SyncService) GitHubClient() githubinterfaces.Client {
	return s.client
}

// FetchNotificationsToSync fetches new notifications from GitHub that need to be synced.
func (s *SyncService) FetchNotificationsToSync(
	ctx context.Context,
) ([]types.NotificationThread, error) {
	state, err := s.syncService.GetSyncState(ctx)
	if err != nil {
		s.logger.Error("failed to get sync state", zap.Error(err))
		return nil, errors.Join(ErrFailedToGetSyncState, err)
	}

	hasState := state.LastSuccessfulPoll.Valid || state.LatestNotificationAt.Valid

	var since *time.Time
	if hasState {
		switch {
		case state.LatestNotificationAt.Valid:
			t := state.LatestNotificationAt.Time
			since = &t
		case state.LastSuccessfulPoll.Valid:
			t := state.LastSuccessfulPoll.Time
			since = &t
		}
	}

	threads, err := s.client.FetchNotifications(ctx, since)
	if err != nil {
		s.logger.Error("failed to fetch notifications from GitHub", zap.Error(err))
		return nil, errors.Join(ErrFailedToFetchNotifications, err)
	}

	if len(threads) == 0 {
		// Update sync state even when no notifications
		now := s.clock().UTC()
		var latestNotification *time.Time
		if state.LatestNotificationAt.Valid {
			t := state.LatestNotificationAt.Time
			latestNotification = &t
		}
		_, err = s.syncService.UpsertSyncState(ctx, &now, latestNotification)
		if err != nil {
			s.logger.Error("failed to update sync state after empty poll", zap.Error(err))
			return nil, errors.Join(ErrFailedToUpdateSyncState, err)
		}
		return []types.NotificationThread{}, nil
	}

	return threads, nil
}

// UpdateSyncStateAfterProcessing updates the sync state after notifications have been processed.
// This should be called after all notification jobs have been queued or processed.
func (s *SyncService) UpdateSyncStateAfterProcessing(
	ctx context.Context,
	latestUpdate time.Time,
) error {
	now := s.clock().UTC()
	var latestNotification *time.Time
	if !latestUpdate.IsZero() {
		latest := latestUpdate.UTC()
		latestNotification = &latest
	}

	if _, err := s.syncService.UpsertSyncState(ctx, &now, latestNotification); err != nil {
		s.logger.Error("failed to update sync state", zap.Error(err))
		return errors.Join(ErrFailedToUpdateSyncState, err)
	}

	return nil
}

// ProcessNotification handles the complete processing of a single notification.
// This includes upserting the repository, fetching subject details, and upserting the notification.
func (s *SyncService) ProcessNotification(
	ctx context.Context,
	thread types.NotificationThread,
) error {
	// Upsert repository
	rawRepo := thread.Repository.Raw()
	repoParams := db.UpsertRepositoryParams{
		GithubID:       models.SQLNullInt64(thread.Repository.ID),
		NodeID:         models.SQLNullString(thread.Repository.NodeID),
		Name:           thread.Repository.Name,
		FullName:       thread.Repository.FullName,
		OwnerLogin:     models.SQLNullString(thread.Repository.Owner.Login),
		OwnerID:        models.SQLNullInt64(thread.Repository.Owner.ID),
		OwnerAvatarUrl: models.SQLNullString(thread.Repository.Owner.AvatarURL),
		OwnerHtmlUrl:   models.SQLNullString(thread.Repository.Owner.HTMLURL),
		Private:        models.SQLNullBoolPtr(&thread.Repository.Private),
		Description:    models.SQLNullStringPtr(thread.Repository.Description),
		HtmlUrl:        models.SQLNullString(thread.Repository.HTMLURL),
		Fork:           models.SQLNullBoolPtr(&thread.Repository.Fork),
		Visibility:     models.SQLNullStringPtr(thread.Repository.Visibility),
		DefaultBranch:  models.SQLNullStringPtr(thread.Repository.DefaultBranch),
		Archived:       thread.Repository.Archived,
		Disabled:       models.SQLNullBoolPtr(&thread.Repository.Disabled),
		PushedAt:       models.SQLNullTime(thread.Repository.PushedAt),
		CreatedAt:      models.SQLNullTime(thread.Repository.CreatedAt),
		UpdatedAt:      models.SQLNullTime(thread.Repository.UpdatedAt),
		Raw: pqtype.NullRawMessage{
			RawMessage: rawRepo,
			Valid:      len(rawRepo) > 0,
		},
	}

	repo, err := s.repositoryService.UpsertRepository(ctx, repoParams)
	if err != nil {
		s.logger.Error(
			"failed to upsert repository",
			zap.String("fullName", thread.Repository.FullName),
			zap.Error(err),
		)
		return errors.Join(ErrFailedToUpsertRepository, err)
	}

	// Fetch subject details
	var (
		subjectPayload   pqtype.NullRawMessage
		subjectFetchedAt sql.NullTime
	)

	if rawSubject, err := s.client.FetchSubjectRaw(ctx, thread.Subject.URL); err == nil &&
		len(rawSubject) > 0 {
		subjectPayload = pqtype.NullRawMessage{
			RawMessage: rawSubject,
			Valid:      true,
		}
		fetched := s.clock().UTC()
		subjectFetchedAt = models.SQLNullTime(&fetched)
	} else if err != nil {
		// Log but don't fail - subject fetch is optional
		//nolint:lll // Long warning message with multiple zap fields
		s.logger.Warn("failed to fetch subject data (continuing without it)", zap.String("githubID", thread.ID), zap.String("subjectURL", thread.Subject.URL), zap.Error(err))
	}

	// Process pull request metadata if subject is a PullRequest
	var pullRequestID sql.NullInt64
	if strings.EqualFold(thread.Subject.Type, "PullRequest") && subjectPayload.Valid {
		if pr, err := s.upsertPullRequestFromSubject(ctx, repo.ID, subjectPayload.RawMessage); err == nil &&
			pr != nil {
			pullRequestID = sql.NullInt64{Int64: pr.ID, Valid: true}
		} else if err != nil {
			// Log but don't fail - PR metadata is optional
			//nolint:lll // Long warning message with multiple zap fields
			s.logger.Warn("failed to upsert pull request metadata (continuing without it)", zap.String("githubID", thread.ID), zap.Int64("repoID", repo.ID), zap.Error(err))
		}
	}

	// Extract author and subject metadata from subject data
	var authorLogin sql.NullString
	var authorID sql.NullInt64
	var subjectNumber sql.NullInt32
	var subjectState sql.NullString
	var subjectMerged sql.NullBool
	var subjectStateReason sql.NullString
	if subjectPayload.Valid {
		authorLogin, authorID = ExtractAuthorFromSubject(subjectPayload.RawMessage)
		subjectNumber = ExtractSubjectNumber(subjectPayload.RawMessage)
		subjectState = ExtractSubjectState(subjectPayload.RawMessage)
		subjectMerged = ExtractSubjectMerged(subjectPayload.RawMessage)
		subjectStateReason = ExtractSubjectStateReason(subjectPayload.RawMessage)
	}

	// Upsert notification
	notificationParams := db.UpsertNotificationParams{
		GithubID:                thread.ID,
		RepositoryID:            repo.ID,
		PullRequestID:           pullRequestID,
		SubjectType:             thread.Subject.Type,
		SubjectTitle:            thread.Subject.Title,
		SubjectUrl:              models.SQLNullString(thread.Subject.URL),
		SubjectLatestCommentUrl: models.SQLNullString(thread.Subject.LatestCommentURL),
		Reason:                  models.SQLNullString(thread.Reason),
		GithubUnread:            models.SQLNullBoolPtr(&thread.Unread),
		GithubUpdatedAt:         models.SQLNullTime(&thread.UpdatedAt),
		GithubLastReadAt:        models.SQLNullTime(thread.LastReadAt),
		GithubUrl:               models.SQLNullString(thread.URL),
		GithubSubscriptionUrl:   models.SQLNullString(thread.SubscriptionURL),
		Payload: pqtype.NullRawMessage{
			RawMessage: thread.Raw,
			Valid:      len(thread.Raw) > 0,
		},
		SubjectRaw:         subjectPayload,
		SubjectFetchedAt:   subjectFetchedAt,
		AuthorLogin:        authorLogin,
		AuthorID:           authorID,
		SubjectNumber:      subjectNumber,
		SubjectState:       subjectState,
		SubjectMerged:      subjectMerged,
		SubjectStateReason: subjectStateReason,
	}

	if _, err := s.queries.UpsertNotification(ctx, notificationParams); err != nil {
		s.logger.Error(
			"failed to upsert notification",
			zap.String("githubID", thread.ID),
			zap.Error(err),
		)
		return errors.Join(ErrFailedToUpsertNotification, err)
	}

	return nil
}

// upsertPullRequestFromSubject extracts PR data from subject JSON and upserts it to the database.
//

func (s *SyncService) upsertPullRequestFromSubject(
	ctx context.Context,
	repoID int64,
	subjectJSON json.RawMessage,
) (*db.PullRequest, error) {
	prData, err := ExtractPullRequestData(subjectJSON)
	if err != nil {
		s.logger.Error("failed to extract pull request data", zap.Error(err))
		return nil, errors.Join(ErrFailedToExtractPullRequestData, err)
	}

	// Convert extracted data to database params
	params := db.UpsertPullRequestParams{
		RepositoryID: repoID,
		GithubID:     models.SQLNullInt64Ptr(prData.GithubID),
		NodeID:       models.SQLNullStringPtr(prData.NodeID),

		Number:      int32(prData.Number),
		Title:       models.SQLNullStringPtr(prData.Title),
		State:       models.SQLNullStringPtr(prData.State),
		Draft:       models.SQLNullBoolPtr(prData.Draft),
		Merged:      models.SQLNullBoolPtr(prData.Merged),
		AuthorLogin: models.SQLNullStringPtr(prData.AuthorLogin),
		AuthorID:    models.SQLNullInt64Ptr(prData.AuthorID),
		CreatedAt:   models.SQLNullTime(prData.CreatedAt),
		UpdatedAt:   models.SQLNullTime(prData.UpdatedAt),
		ClosedAt:    models.SQLNullTime(prData.ClosedAt),
		MergedAt:    models.SQLNullTime(prData.MergedAt),
		Raw: pqtype.NullRawMessage{
			RawMessage: subjectJSON,
			Valid:      true,
		},
	}

	pr, err := s.queries.UpsertPullRequest(ctx, params)
	if err != nil {
		s.logger.Error(
			"failed to upsert pull request",
			zap.Int64("repoID", repoID),
			zap.Int("number", prData.Number),
			zap.Error(err),
		)
		return nil, errors.Join(ErrFailedToUpsertPullRequest, err)
	}

	return &pr, nil
}

// RefreshSubjectData fetches fresh subject data from GitHub and updates the notification
func (s *SyncService) RefreshSubjectData(ctx context.Context, githubID string) error {
	// Get the notification
	notification, err := s.queries.GetNotificationByGithubID(ctx, githubID)
	if err != nil {
		s.logger.Error(
			"failed to get notification",
			zap.String("githubID", githubID),
			zap.Error(err),
		)
		return errors.Join(ErrFailedToGetNotification, err)
	}

	if !notification.SubjectUrl.Valid || notification.SubjectUrl.String == "" {
		s.logger.Warn("notification has no subject URL", zap.String("githubID", githubID))
		return ErrNotificationMissingSubjectURL
	}

	// Fetch fresh subject data
	subjectRaw, err := s.client.FetchSubjectRaw(ctx, notification.SubjectUrl.String)
	if err != nil {
		s.logger.Error(
			"failed to fetch subject from GitHub",
			zap.String("githubID", githubID),
			zap.String("subjectURL", notification.SubjectUrl.String),
			zap.Error(err),
		)
		return errors.Join(ErrFailedToFetchSubject, err)
	}

	// Update the notification with fresh subject data
	subjectPayload := pqtype.NullRawMessage{
		RawMessage: subjectRaw,
		Valid:      len(subjectRaw) > 0,
	}
	fetched := s.clock().UTC()
	subjectFetchedAt := models.SQLNullTime(&fetched)

	// If it's a PR, update the pull_request table too
	var pullRequestID sql.NullInt64
	if strings.EqualFold(notification.SubjectType, "PullRequest") && subjectPayload.Valid {
		repo, repoErr := s.repositoryService.GetRepositoryByID(ctx, notification.RepositoryID)
		if repoErr != nil {
			s.logger.Error(
				"failed to get repository",
				zap.Int64("repositoryID", notification.RepositoryID),
				zap.Error(repoErr),
			)
			return errors.Join(ErrFailedToGetRepository, repoErr)
		}

		if pr, prErr := s.upsertPullRequestFromSubject(ctx, repo.ID, subjectPayload.RawMessage); prErr == nil &&
			pr != nil {
			pullRequestID = sql.NullInt64{Int64: pr.ID, Valid: true}
		} else if prErr != nil {
			// Log but don't fail - PR metadata update is optional
			s.logger.Warn(
				"failed to upsert pull request metadata during refresh (continuing without it)",
				zap.String("githubID", githubID),
				zap.Int64("repoID", repo.ID),
				zap.Error(prErr),
			)
		}
	}

	// Extract subject metadata from fresh subject data
	var subjectNumber sql.NullInt32
	var subjectState sql.NullString
	var subjectMerged sql.NullBool
	var subjectStateReason sql.NullString
	if subjectPayload.Valid {
		subjectNumber = ExtractSubjectNumber(subjectPayload.RawMessage)
		subjectState = ExtractSubjectState(subjectPayload.RawMessage)
		subjectMerged = ExtractSubjectMerged(subjectPayload.RawMessage)
		subjectStateReason = ExtractSubjectStateReason(subjectPayload.RawMessage)
	}

	// Update the notification with the fresh subject data
	err = s.queries.UpdateNotificationSubject(ctx, db.UpdateNotificationSubjectParams{
		GithubID:           githubID,
		SubjectRaw:         subjectPayload,
		SubjectFetchedAt:   subjectFetchedAt,
		PullRequestID:      pullRequestID,
		SubjectNumber:      subjectNumber,
		SubjectState:       subjectState,
		SubjectMerged:      subjectMerged,
		SubjectStateReason: subjectStateReason,
	})
	if err != nil {
		s.logger.Error(
			"failed to update notification subject",
			zap.String("githubID", githubID),
			zap.Error(err),
		)
		return errors.Join(ErrFailedToUpdateNotificationSubject, err)
	}

	return nil
}
