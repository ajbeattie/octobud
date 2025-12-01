-- name: GetSyncState :one
SELECT id,
       last_successful_poll,
       latest_notification_at,
       last_notification_etag,
       created_at,
       updated_at
FROM sync_state
WHERE id = 1;

-- name: UpdateSyncState :one
UPDATE sync_state
SET last_successful_poll  = sqlc.narg('last_successful_poll'),
    latest_notification_at = sqlc.narg('latest_notification_at'),
    last_notification_etag = sqlc.narg('last_notification_etag'),
    updated_at = now()
WHERE id = 1
RETURNING id,
          last_successful_poll,
          latest_notification_at,
          last_notification_etag,
          created_at,
          updated_at;

-- name: UpsertSyncState :one
INSERT INTO sync_state (id, last_successful_poll, latest_notification_at, last_notification_etag)
VALUES (1, sqlc.narg('last_successful_poll'), sqlc.narg('latest_notification_at'), sqlc.narg('last_notification_etag'))
ON CONFLICT (id) DO UPDATE
SET last_successful_poll  = EXCLUDED.last_successful_poll,
    latest_notification_at = EXCLUDED.latest_notification_at,
    last_notification_etag = EXCLUDED.last_notification_etag,
    updated_at = now()
RETURNING id,
          last_successful_poll,
          latest_notification_at,
          last_notification_etag,
          created_at,
          updated_at;

