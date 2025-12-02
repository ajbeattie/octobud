-- +goose Up
CREATE TABLE IF NOT EXISTS pull_requests (
    id BIGSERIAL PRIMARY KEY,
    repository_id BIGINT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    github_id BIGINT,
    node_id TEXT,
    number INTEGER NOT NULL,
    title TEXT,
    state TEXT,
    draft BOOLEAN,
    merged BOOLEAN,
    author_login TEXT,
    author_id BIGINT,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    closed_at TIMESTAMPTZ,
    merged_at TIMESTAMPTZ,
    raw JSONB,
    UNIQUE (repository_id, number)
);

-- +goose Down
DROP TABLE IF EXISTS pull_requests;

