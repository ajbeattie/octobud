-- +goose Up
ALTER TABLE users ADD COLUMN IF NOT EXISTS sync_settings JSONB;

-- +goose Down
ALTER TABLE users DROP COLUMN IF EXISTS sync_settings;

