-- name: UpsertNotification :one
INSERT INTO notifications (
    github_id,
    repository_id,
    pull_request_id,
    subject_type,
    subject_title,
    subject_url,
    subject_latest_comment_url,
    reason,
    github_unread,
    github_updated_at,
    github_last_read_at,
    github_url,
    github_subscription_url,
    payload,
    subject_raw,
    subject_fetched_at,
    author_login,
    author_id,
    subject_number,
    subject_state,
    subject_merged,
    subject_state_reason,
    effective_sort_date
)
VALUES (
    sqlc.arg('github_id'),
    sqlc.arg('repository_id'),
    sqlc.narg('pull_request_id'),
    sqlc.arg('subject_type'),
    sqlc.arg('subject_title'),
    sqlc.narg('subject_url'),
    sqlc.narg('subject_latest_comment_url'),
    sqlc.narg('reason'),
    sqlc.narg('github_unread'),
    sqlc.narg('github_updated_at'),
    sqlc.narg('github_last_read_at'),
    sqlc.narg('github_url'),
    sqlc.narg('github_subscription_url'),
    sqlc.narg('payload'),
    sqlc.narg('subject_raw'),
    sqlc.narg('subject_fetched_at'),
    sqlc.narg('author_login'),
    sqlc.narg('author_id'),
    sqlc.narg('subject_number'),
    sqlc.narg('subject_state'),
    sqlc.narg('subject_merged'),
    sqlc.narg('subject_state_reason'),
    sqlc.narg('github_updated_at')
)
ON CONFLICT (github_id) DO UPDATE
SET repository_id = EXCLUDED.repository_id,
    pull_request_id = EXCLUDED.pull_request_id,
    subject_type = EXCLUDED.subject_type,
    subject_title = EXCLUDED.subject_title,
    subject_url = EXCLUDED.subject_url,
    subject_latest_comment_url = EXCLUDED.subject_latest_comment_url,
    reason = EXCLUDED.reason,
    github_unread = EXCLUDED.github_unread,
    github_updated_at = EXCLUDED.github_updated_at,
    github_last_read_at = EXCLUDED.github_last_read_at,
    github_url = EXCLUDED.github_url,
    github_subscription_url = EXCLUDED.github_subscription_url,
    payload = EXCLUDED.payload,
    subject_raw = EXCLUDED.subject_raw,
    subject_fetched_at = EXCLUDED.subject_fetched_at,
    author_login = EXCLUDED.author_login,
    author_id = EXCLUDED.author_id,
    subject_number = EXCLUDED.subject_number,
    subject_state = EXCLUDED.subject_state,
    subject_merged = EXCLUDED.subject_merged,
    subject_state_reason = EXCLUDED.subject_state_reason,
    imported_at = now(),
    -- Smart status updates on sync
    is_read = CASE
        WHEN notifications.muted THEN notifications.is_read
        WHEN EXCLUDED.github_updated_at IS DISTINCT FROM notifications.github_updated_at
        THEN false
        ELSE notifications.is_read
    END,
    archived = CASE
        WHEN notifications.muted THEN notifications.archived
        WHEN EXCLUDED.github_updated_at IS DISTINCT FROM notifications.github_updated_at
        THEN false
        ELSE notifications.archived
    END,
    -- Preserve snoozed_until: time-based snooze, not update-triggered
    snoozed_until = notifications.snoozed_until,
    -- Always preserve muted status.
    muted = notifications.muted,
    -- Always preserve filtered status (managed by rules).
    filtered = notifications.filtered,
    -- Update effective_sort_date: use existing snoozed_until if set, otherwise use new github_updated_at
    effective_sort_date = COALESCE(notifications.snoozed_until, EXCLUDED.github_updated_at)
RETURNING *;

-- name: GetNotificationByGithubID :one
SELECT *
FROM notifications
WHERE github_id = sqlc.arg('github_id');

-- name: GetNotificationByID :one
SELECT *
FROM notifications
WHERE id = sqlc.arg('id');

-- name: MarkNotificationRead :one
UPDATE notifications
SET is_read = true
WHERE github_id = sqlc.arg('github_id')
RETURNING *;

-- name: MarkNotificationUnread :one
UPDATE notifications
SET is_read = false
WHERE github_id = sqlc.arg('github_id')
RETURNING *;

-- name: BulkMarkNotificationsRead :execrows
UPDATE notifications
SET is_read = true
WHERE github_id = ANY(sqlc.arg('github_ids')::text[]);

-- name: BulkMarkNotificationsUnread :execrows
UPDATE notifications
SET is_read = false
WHERE github_id = ANY(sqlc.arg('github_ids')::text[]);

-- name: BulkArchiveNotifications :execrows
UPDATE notifications
SET archived = true,
    snoozed_until = NULL,
    snoozed_at = NULL,
    effective_sort_date = COALESCE(github_updated_at, imported_at)
WHERE github_id = ANY(sqlc.arg('github_ids')::text[]);

-- name: BulkUnarchiveNotifications :execrows
UPDATE notifications
SET archived = false
WHERE github_id = ANY(sqlc.arg('github_ids')::text[]);

-- name: BulkSnoozeNotifications :execrows
UPDATE notifications
SET snoozed_until = sqlc.arg('snoozed_until'),
    snoozed_at = NOW(),
    effective_sort_date = sqlc.arg('snoozed_until')
WHERE github_id = ANY(sqlc.arg('github_ids')::text[]);

-- name: MuteNotification :one
UPDATE notifications
SET muted = true,
    snoozed_until = NULL,
    snoozed_at = NULL,
    effective_sort_date = COALESCE(github_updated_at, imported_at)
WHERE github_id = sqlc.arg('github_id')
RETURNING *;

-- name: UnmuteNotification :one
UPDATE notifications
SET muted = false
WHERE github_id = sqlc.arg('github_id')
RETURNING *;

-- name: BulkMuteNotifications :execrows
UPDATE notifications
SET muted = true,
    snoozed_until = NULL,
    snoozed_at = NULL,
    effective_sort_date = COALESCE(github_updated_at, imported_at)
WHERE github_id = ANY(sqlc.arg('github_ids')::text[]);

-- name: BulkUnmuteNotifications :execrows
UPDATE notifications
SET muted = false
WHERE github_id = ANY(sqlc.arg('github_ids')::text[]);

-- name: SnoozeNotification :one
UPDATE notifications
SET snoozed_until = sqlc.arg('snoozed_until'),
    snoozed_at = NOW(),
    effective_sort_date = sqlc.arg('snoozed_until')
WHERE github_id = sqlc.arg('github_id')
RETURNING *;

-- name: UnsnoozeNotification :one
UPDATE notifications
SET snoozed_until = NULL,
    snoozed_at = NULL,
    effective_sort_date = COALESCE(github_updated_at, imported_at)
WHERE github_id = sqlc.arg('github_id')
RETURNING *;

-- name: BulkUnsnoozeNotifications :execrows
UPDATE notifications
SET snoozed_until = NULL,
    snoozed_at = NULL,
    effective_sort_date = COALESCE(github_updated_at, imported_at)
WHERE github_id = ANY(sqlc.arg('github_ids')::text[]);

-- name: ListNotifications :many
SELECT *
FROM notifications
ORDER BY github_updated_at DESC NULLS LAST, imported_at DESC;

-- name: ListNotificationsForRepository :many
SELECT *
FROM notifications
WHERE repository_id = sqlc.arg('repository_id')
ORDER BY github_updated_at DESC NULLS LAST, imported_at DESC;

-- name: ArchiveNotification :one
UPDATE notifications
SET archived = TRUE,
    snoozed_until = NULL,
    snoozed_at = NULL,
    effective_sort_date = COALESCE(github_updated_at, imported_at)
WHERE github_id = sqlc.arg('github_id')
RETURNING *;

-- name: UnarchiveNotification :one
UPDATE notifications
SET archived = FALSE
WHERE github_id = sqlc.arg('github_id')
RETURNING *;

-- name: UpdateNotificationSubject :exec
UPDATE notifications
SET subject_raw = sqlc.narg('subject_raw'),
    subject_fetched_at = sqlc.narg('subject_fetched_at'),
    pull_request_id = sqlc.narg('pull_request_id'),
    subject_number = sqlc.narg('subject_number'),
    subject_state = sqlc.narg('subject_state'),
    subject_merged = sqlc.narg('subject_merged'),
    subject_state_reason = sqlc.narg('subject_state_reason')
WHERE github_id = sqlc.arg('github_id');

-- name: StarNotification :one
UPDATE notifications
SET starred = TRUE
WHERE github_id = sqlc.arg('github_id')
RETURNING *;

-- name: UnstarNotification :one
UPDATE notifications
SET starred = FALSE
WHERE github_id = sqlc.arg('github_id')
RETURNING *;

-- name: BulkStarNotifications :execrows
UPDATE notifications
SET starred = TRUE
WHERE github_id = ANY(sqlc.arg('github_ids')::text[]);

-- name: BulkUnstarNotifications :execrows
UPDATE notifications
SET starred = FALSE
WHERE github_id = ANY(sqlc.arg('github_ids')::text[]);

-- name: MarkNotificationFiltered :one
UPDATE notifications
SET filtered = TRUE
WHERE github_id = sqlc.arg('github_id')
RETURNING *;

-- name: MarkNotificationUnfiltered :one
UPDATE notifications
SET filtered = FALSE
WHERE github_id = sqlc.arg('github_id')
RETURNING *;

-- name: BulkMarkNotificationsFiltered :execrows
UPDATE notifications
SET filtered = TRUE
WHERE github_id = ANY(sqlc.arg('github_ids')::text[]);

-- name: BulkMarkNotificationsUnfiltered :execrows
UPDATE notifications
SET filtered = FALSE
WHERE github_id = ANY(sqlc.arg('github_ids')::text[]);

