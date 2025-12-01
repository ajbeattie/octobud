-- +goose Up
CREATE TABLE IF NOT EXISTS tags (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    color TEXT,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    display_order INTEGER NOT NULL DEFAULT 0,
    slug TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_tags_display_order ON tags(display_order);
CREATE UNIQUE INDEX IF NOT EXISTS idx_tags_slug ON tags(slug);

-- +goose Down
DROP TABLE IF EXISTS tags;

