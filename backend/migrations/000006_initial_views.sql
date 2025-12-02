-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS views (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    icon TEXT,
    slug TEXT NOT NULL DEFAULT gen_random_uuid()::text,
    query TEXT,
    display_order INTEGER DEFAULT 0 NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS views_slug_unique ON views(slug);

-- +goose Down
DROP TABLE IF EXISTS views;
DROP EXTENSION IF EXISTS pgcrypto;

