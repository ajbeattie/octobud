-- +goose Up
CREATE TABLE IF NOT EXISTS sync_state (
    id INTEGER PRIMARY KEY DEFAULT 1,
    last_successful_poll TIMESTAMPTZ,
    last_notification_etag TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    latest_notification_at TIMESTAMPTZ
);

INSERT INTO sync_state (id) VALUES (1)
ON CONFLICT (id) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS sync_state;

