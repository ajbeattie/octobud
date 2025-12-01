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

// Package interfaces defines interfaces for GitHub operations.
// This package is separate to avoid import cycles when generating mocks.
package interfaces

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ajbeattie/octobud/backend/internal/github/types"
)

// Client defines the interface for GitHub API operations.
type Client interface {
	SetToken(ctx context.Context, token string) error
	FetchNotifications(ctx context.Context, since *time.Time) ([]types.NotificationThread, error)
	FetchSubjectRaw(ctx context.Context, subjectURL string) (json.RawMessage, error)
	FetchTimeline(
		ctx context.Context,
		owner, repo string,
		number, perPage, page int,
	) ([]types.TimelineEvent, error)
	FetchIssueComments(
		ctx context.Context,
		owner, repo string,
		number, perPage, page int,
	) ([]types.IssueComment, error)
	FetchPullRequestReviews(
		ctx context.Context,
		owner, repo string,
		number, perPage, page int,
	) ([]types.PullRequestReview, error)
}
