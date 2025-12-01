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

// Package db provides the database queries for the application.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/lib/pq"
)

// NotificationQuery represents a complete notification query ready for SQL execution
type NotificationQuery struct {
	Joins          []string      // SQL JOIN clauses needed
	Where          []string      // SQL WHERE conditions
	Args           []interface{} // Query parameters for prepared statements
	Limit          int32
	Offset         int32
	IncludeSubject bool // Whether to include subject_raw in SELECT (default: true for backward compatibility)
}

// notificationColumns returns the list of all notification table columns in order.
// This must match the order used in rows.Scan() below.
// When adding new columns via migrations, update this function and the corresponding Scan() calls.
// Column order (as of latest migration):
// 0: id, 1: github_id, 2: repository_id, 3: pull_request_id, 4: subject_type, 5: subject_title,
// 6: subject_url, 7: subject_latest_comment_url, 8: reason, 9: archived, 10: github_unread,
// 11: github_updated_at, 12: github_last_read_at, 13: github_url, 14: github_subscription_url,
// 15: imported_at, 16: payload, 17: subject_raw, 18: subject_fetched_at, 19: author_login,
// 20: author_id, 21: is_read, 22: muted, 23: snoozed_until, 24: effective_sort_date,
// 25: snoozed_at, 26: starred, 27: filtered, 28: tag_ids, 29: subject_number, 30: subject_state,
// 31: subject_merged, 32: subject_state_reason
func notificationColumns(includeSubject bool) string {
	columns := []string{
		"n.id",                         // 0
		"n.github_id",                  // 1
		"n.repository_id",              // 2
		"n.pull_request_id",            // 3
		"n.subject_type",               // 4
		"n.subject_title",              // 5
		"n.subject_url",                // 6
		"n.subject_latest_comment_url", // 7
		"n.reason",                     // 8
		"n.archived",                   // 9
		"n.github_unread",              // 10
		"n.github_updated_at",          // 11
		"n.github_last_read_at",        // 12
		"n.github_url",                 // 13
		"n.github_subscription_url",    // 14
		"n.imported_at",                // 15
		"n.payload",                    // 16
		"n.subject_fetched_at",         // 17
		"n.author_login",               // 18
		"n.author_id",                  // 19
		"n.is_read",                    // 20
		"n.muted",                      // 21
		"n.snoozed_until",              // 22
		"n.effective_sort_date",        // 23
		"n.snoozed_at",                 // 24
		"n.starred",                    // 25
		"n.filtered",                   // 26
		"n.tag_ids",                    // 27
		"n.subject_number",             // 28
		"n.subject_state",              // 29
		"n.subject_merged",             // 30
		"n.subject_state_reason",       // 31
	}

	// If includeSubject is true, add subject_raw to the columns.
	if includeSubject {
		columns = append(columns, "n.subject_raw")
	}

	return "SELECT " + strings.Join(columns, ", ") + " FROM notifications n"
}

// ListNotificationsFromQueryResult contains the notifications and total count
type ListNotificationsFromQueryResult struct {
	Notifications []Notification
	Total         int64
}

// ListNotificationsFromQuery executes a notification query built by the query builder
func (q *Queries) ListNotificationsFromQuery(
	ctx context.Context,
	query NotificationQuery,
) (ListNotificationsFromQueryResult, error) {
	// Build the SELECT query - conditionally exclude subject_raw to reduce data transfer
	baseSelect := notificationColumns(query.IncludeSubject)

	// Add JOINs
	joins := ""
	if len(query.Joins) > 0 {
		joins = " " + strings.Join(query.Joins, " ")
	}

	// Add WHERE clause
	where := ""
	if len(query.Where) > 0 {
		where = " WHERE " + strings.Join(query.Where, " AND ")
	}

	// Add ORDER BY
	// Sort by effective_sort_date which is managed by the application layer
	orderBy := " ORDER BY n.effective_sort_date DESC NULLS LAST, n.imported_at DESC"

	// Add LIMIT and OFFSET
	limitOffset := fmt.Sprintf(" LIMIT %d OFFSET %d", query.Limit, query.Offset)

	// Combine everything
	selectQuery := baseSelect + joins + where + orderBy + limitOffset

	// Execute query
	rows, err := q.db.QueryContext(ctx, selectQuery, query.Args...)
	if err != nil {
		return ListNotificationsFromQueryResult{}, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var notifications []Notification
	for rows.Next() {
		var n Notification
		// Column order must match notificationColumns() function above
		// When adding new columns, update both notificationColumns() and these Scan() calls
		scanColumns := []any{
			&n.ID,                      // 0
			&n.GithubID,                // 1
			&n.RepositoryID,            // 2
			&n.PullRequestID,           // 3
			&n.SubjectType,             // 4
			&n.SubjectTitle,            // 5
			&n.SubjectUrl,              // 6
			&n.SubjectLatestCommentUrl, // 7
			&n.Reason,                  // 8
			&n.Archived,                // 9
			&n.GithubUnread,            // 10
			&n.GithubUpdatedAt,         // 11
			&n.GithubLastReadAt,        // 12
			&n.GithubUrl,               // 13
			&n.GithubSubscriptionUrl,   // 14
			&n.ImportedAt,              // 15
			&n.Payload,                 // 16
			&n.SubjectFetchedAt,        // 17
			&n.AuthorLogin,             // 18
			&n.AuthorID,                // 19
			&n.IsRead,                  // 20
			&n.Muted,                   // 21
			&n.SnoozedUntil,            // 22
			&n.EffectiveSortDate,       // 23
			&n.SnoozedAt,               // 24
			&n.Starred,                 // 25
			&n.Filtered,                // 26
			pq.Array(&n.TagIds),        // 27
			&n.SubjectNumber,           // 28
			&n.SubjectState,            // 29
			&n.SubjectMerged,           // 30
			&n.SubjectStateReason,      // 31
		}

		// For convience, add subject_raw and any other future optional columns last so that
		// the static columns don't have their order changed by new columns.
		if query.IncludeSubject {
			scanColumns = append(scanColumns, &n.SubjectRaw)
		}

		// Scan all columns in the order of the scanColumns array.
		scanErr := rows.Scan(
			scanColumns...,
		)
		if scanErr != nil {
			return ListNotificationsFromQueryResult{}, scanErr
		}

		notifications = append(notifications, n)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		return ListNotificationsFromQueryResult{}, rowsErr
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM notifications n" + joins + where
	var total int64
	err = q.db.QueryRowContext(ctx, countQuery, query.Args...).Scan(&total)
	if err != nil {
		return ListNotificationsFromQueryResult{}, err
	}

	return ListNotificationsFromQueryResult{
		Notifications: notifications,
		Total:         total,
	}, nil
}

// BulkMarkNotificationsReadByQuery marks all notifications matching a query as read
func (q *Queries) BulkMarkNotificationsReadByQuery(
	ctx context.Context,
	query NotificationQuery,
) (int64, error) {
	return q.executeBulkUpdateByQuery(ctx, query, "UPDATE notifications n SET is_read = TRUE")
}

// BulkMarkNotificationsUnreadByQuery marks all notifications matching a query as unread
func (q *Queries) BulkMarkNotificationsUnreadByQuery(
	ctx context.Context,
	query NotificationQuery,
) (int64, error) {
	return q.executeBulkUpdateByQuery(ctx, query, "UPDATE notifications n SET is_read = FALSE")
}

// BulkArchiveNotificationsByQuery archives all notifications matching a query
func (q *Queries) BulkArchiveNotificationsByQuery(
	ctx context.Context,
	query NotificationQuery,
) (int64, error) {
	return q.executeBulkUpdateByQuery(
		ctx,
		query,
		"UPDATE notifications n SET archived = TRUE, snoozed_until = NULL, "+
			"snoozed_at = NULL, effective_sort_date = COALESCE(n.github_updated_at, n.imported_at)",
	)
}

// BulkUnarchiveNotificationsByQuery unarchives all notifications matching a query
func (q *Queries) BulkUnarchiveNotificationsByQuery(
	ctx context.Context,
	query NotificationQuery,
) (int64, error) {
	return q.executeBulkUpdateByQuery(ctx, query, "UPDATE notifications n SET archived = FALSE")
}

// BulkSnoozeNotificationsByQueryParams contains the parameters for snoozing by query
type BulkSnoozeNotificationsByQueryParams struct {
	Query        NotificationQuery
	SnoozedUntil sql.NullTime
}

// BulkSnoozeNotificationsByQuery snoozes all notifications matching a query
func (q *Queries) BulkSnoozeNotificationsByQuery(
	ctx context.Context,
	arg BulkSnoozeNotificationsByQueryParams,
) (int64, error) {
	// For UPDATE with JOINs in PostgreSQL, we need to use a subquery approach
	// First, build a SELECT query to get the IDs of notifications matching the criteria
	selectQuery := "SELECT n.github_id FROM notifications n"

	// Add JOINs
	if len(arg.Query.Joins) > 0 {
		selectQuery += " " + strings.Join(arg.Query.Joins, " ")
	}

	// Add WHERE clause
	if len(arg.Query.Where) > 0 {
		selectQuery += " WHERE " + strings.Join(arg.Query.Where, " AND ")
	}

	// Increment all placeholder numbers in the selectQuery by 1 to make room for the snoozedUntil parameter at $1
	selectQuery = incrementPlaceholders(selectQuery, 1)

	// Build the UPDATE query with the snoozedUntil parameter
	baseUpdate := "UPDATE notifications n SET snoozed_until = $1, snoozed_at = NOW(), effective_sort_date = $1"
	updateQuery := baseUpdate + " WHERE n.github_id IN (" + selectQuery + ")"

	// Prepend snoozedUntil to args
	args := append([]interface{}{arg.SnoozedUntil}, arg.Query.Args...)

	// Execute query
	result, err := q.db.ExecContext(ctx, updateQuery, args...)
	if err != nil {
		return 0, err
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return count, nil
}

// BulkUnsnoozeNotificationsByQuery unsnoozes all notifications matching a query
func (q *Queries) BulkUnsnoozeNotificationsByQuery(
	ctx context.Context,
	query NotificationQuery,
) (int64, error) {
	updateSQL := "UPDATE notifications n SET snoozed_until = NULL, snoozed_at = NULL, " +
		"effective_sort_date = COALESCE(n.github_updated_at, n.imported_at)"
	return q.executeBulkUpdateByQuery(ctx, query, updateSQL)
}

// BulkMuteNotificationsByQuery mutes all notifications matching a query
func (q *Queries) BulkMuteNotificationsByQuery(
	ctx context.Context,
	query NotificationQuery,
) (int64, error) {
	return q.executeBulkUpdateByQuery(
		ctx,
		query,
		"UPDATE notifications n SET muted = TRUE, snoozed_until = NULL, snoozed_at = NULL, "+
			"effective_sort_date = COALESCE(n.github_updated_at, n.imported_at)",
	)
}

// BulkUnmuteNotificationsByQuery unmutes all notifications matching a query
func (q *Queries) BulkUnmuteNotificationsByQuery(
	ctx context.Context,
	query NotificationQuery,
) (int64, error) {
	return q.executeBulkUpdateByQuery(ctx, query, "UPDATE notifications n SET muted = FALSE")
}

// BulkStarNotificationsByQuery stars all notifications matching a query
func (q *Queries) BulkStarNotificationsByQuery(
	ctx context.Context,
	query NotificationQuery,
) (int64, error) {
	return q.executeBulkUpdateByQuery(ctx, query, "UPDATE notifications n SET starred = TRUE")
}

// BulkUnstarNotificationsByQuery unstars all notifications matching a query
func (q *Queries) BulkUnstarNotificationsByQuery(
	ctx context.Context,
	query NotificationQuery,
) (int64, error) {
	return q.executeBulkUpdateByQuery(ctx, query, "UPDATE notifications n SET starred = FALSE")
}

// executeBulkUpdateByQuery executes a bulk UPDATE operation based on a query
func (q *Queries) executeBulkUpdateByQuery(
	ctx context.Context,
	query NotificationQuery,
	baseUpdate string,
) (int64, error) {
	// For UPDATE with JOINs in PostgreSQL, we need to use a subquery approach
	// First, build a SELECT query to get the IDs of notifications matching the criteria
	selectQuery := "SELECT n.github_id FROM notifications n"

	// Add JOINs
	if len(query.Joins) > 0 {
		selectQuery += " " + strings.Join(query.Joins, " ")
	}

	// Add WHERE clause
	if len(query.Where) > 0 {
		selectQuery += " WHERE " + strings.Join(query.Where, " AND ")
	}

	// Now build the UPDATE query using the subquery
	updateQuery := baseUpdate + " WHERE n.github_id IN (" + selectQuery + ")"

	// Execute query
	result, err := q.db.ExecContext(ctx, updateQuery, query.Args...)
	if err != nil {
		return 0, err
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return count, nil
}

// incrementPlaceholders increments all placeholder numbers in a SQL string by offset
// For example, incrementPlaceholders("WHERE x = $1 AND y = $2", 1) returns "WHERE x = $2 AND y = $3"
func incrementPlaceholders(sql string, offset int) string {
	if offset == 0 {
		return sql
	}

	// Find all placeholders in the format $N
	var result strings.Builder
	i := 0
	for i < len(sql) {
		if sql[i] == '$' && i+1 < len(sql) {
			// Found a potential placeholder
			j := i + 1
			for j < len(sql) && sql[j] >= '0' && sql[j] <= '9' {
				j++
			}
			if j > i+1 {
				// Extract the number
				numStr := sql[i+1 : j]
				var num int

				_, _ = fmt.Sscanf(numStr, "%d", &num)
				// Write the incremented placeholder
				result.WriteString(fmt.Sprintf("$%d", num+offset))
				i = j
				continue
			}
		}
		result.WriteByte(sql[i])
		i++
	}
	return result.String()
}
