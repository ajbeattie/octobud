-- name: UpsertPullRequest :one
INSERT INTO pull_requests (
    repository_id,
    github_id,
    node_id,
    number,
    title,
    state,
    draft,
    merged,
    author_login,
    author_id,
    created_at,
    updated_at,
    closed_at,
    merged_at,
    raw
)
VALUES (
    sqlc.arg('repository_id'),
    sqlc.narg('github_id'),
    sqlc.narg('node_id'),
    sqlc.arg('number'),
    sqlc.narg('title'),
    sqlc.narg('state'),
    sqlc.narg('draft'),
    sqlc.narg('merged'),
    sqlc.narg('author_login'),
    sqlc.narg('author_id'),
    sqlc.narg('created_at'),
    sqlc.narg('updated_at'),
    sqlc.narg('closed_at'),
    sqlc.narg('merged_at'),
    sqlc.narg('raw')
)
ON CONFLICT (repository_id, number) DO UPDATE
SET github_id = COALESCE(EXCLUDED.github_id, pull_requests.github_id),
    node_id = COALESCE(EXCLUDED.node_id, pull_requests.node_id),
    title = COALESCE(EXCLUDED.title, pull_requests.title),
    state = COALESCE(EXCLUDED.state, pull_requests.state),
    draft = COALESCE(EXCLUDED.draft, pull_requests.draft),
    merged = COALESCE(EXCLUDED.merged, pull_requests.merged),
    author_login = COALESCE(EXCLUDED.author_login, pull_requests.author_login),
    author_id = COALESCE(EXCLUDED.author_id, pull_requests.author_id),
    created_at = COALESCE(EXCLUDED.created_at, pull_requests.created_at),
    updated_at = COALESCE(EXCLUDED.updated_at, pull_requests.updated_at),
    closed_at = COALESCE(EXCLUDED.closed_at, pull_requests.closed_at),
    merged_at = COALESCE(EXCLUDED.merged_at, pull_requests.merged_at),
    raw = COALESCE(EXCLUDED.raw, pull_requests.raw)
RETURNING *;

-- name: GetPullRequestByRepositoryAndNumber :one
SELECT *
FROM pull_requests
WHERE repository_id = sqlc.arg('repository_id')
  AND number = sqlc.arg('number');

-- name: GetPullRequestByID :one
SELECT *
FROM pull_requests
WHERE id = sqlc.arg('id');

