-- +goose Up
CREATE TABLE IF NOT EXISTS repositories (
    id BIGSERIAL PRIMARY KEY,
    github_id BIGINT,
    node_id TEXT,
    name TEXT NOT NULL,
    full_name TEXT NOT NULL UNIQUE,
    owner_login TEXT,
    owner_id BIGINT,
    private BOOLEAN,
    description TEXT,
    html_url TEXT,
    fork BOOLEAN,
    visibility TEXT,
    default_branch TEXT,
    archived BOOLEAN DEFAULT FALSE,
    disabled BOOLEAN DEFAULT FALSE,
    pushed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    raw JSONB,
    owner_avatar_url TEXT,
    owner_html_url TEXT
);

-- +goose Down
DROP TABLE IF EXISTS repositories;

