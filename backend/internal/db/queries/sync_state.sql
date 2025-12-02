-- name: GetSyncState :one
SELECT id,
       last_successful_poll,
       latest_notification_at,
       last_notification_etag,
       created_at,
       updated_at,
       initial_sync_completed_at,
       oldest_notification_synced_at
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
INSERT INTO sync_state (id, last_successful_poll, latest_notification_at, last_notification_etag, initial_sync_completed_at, oldest_notification_synced_at)
VALUES (1, sqlc.narg('last_successful_poll'), sqlc.narg('latest_notification_at'), sqlc.narg('last_notification_etag'), sqlc.narg('initial_sync_completed_at'), sqlc.narg('oldest_notification_synced_at'))
ON CONFLICT (id) DO UPDATE
SET last_successful_poll  = COALESCE(EXCLUDED.last_successful_poll, sync_state.last_successful_poll),
    latest_notification_at = COALESCE(EXCLUDED.latest_notification_at, sync_state.latest_notification_at),
    last_notification_etag = COALESCE(EXCLUDED.last_notification_etag, sync_state.last_notification_etag),
    initial_sync_completed_at = COALESCE(EXCLUDED.initial_sync_completed_at, sync_state.initial_sync_completed_at),
    oldest_notification_synced_at = COALESCE(EXCLUDED.oldest_notification_synced_at, sync_state.oldest_notification_synced_at),
    updated_at = now()
RETURNING id,
          last_successful_poll,
          latest_notification_at,
          last_notification_etag,
          created_at,
          updated_at,
          initial_sync_completed_at,
          oldest_notification_synced_at;

