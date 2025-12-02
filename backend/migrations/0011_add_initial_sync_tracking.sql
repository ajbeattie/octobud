-- +goose Up
ALTER TABLE sync_state ADD COLUMN IF NOT EXISTS initial_sync_completed_at TIMESTAMPTZ;
ALTER TABLE sync_state ADD COLUMN IF NOT EXISTS oldest_notification_synced_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE sync_state DROP COLUMN IF EXISTS initial_sync_completed_at;
ALTER TABLE sync_state DROP COLUMN IF EXISTS oldest_notification_synced_at;

