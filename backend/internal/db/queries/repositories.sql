-- name: UpsertRepository :one
INSERT INTO repositories (
    github_id,
    node_id,
    name,
    full_name,
    owner_login,
    owner_id,
    private,
    description,
    html_url,
    owner_avatar_url,
    owner_html_url,
    fork,
    visibility,
    default_branch,
    archived,
    disabled,
    pushed_at,
    created_at,
    updated_at,
    raw
)
VALUES (
    sqlc.narg('github_id'),
    sqlc.narg('node_id'),
    sqlc.arg('name'),
    sqlc.arg('full_name'),
    sqlc.narg('owner_login'),
    sqlc.narg('owner_id'),
    sqlc.narg('private'),
    sqlc.narg('description'),
    sqlc.narg('html_url'),
    sqlc.narg('owner_avatar_url'),
    sqlc.narg('owner_html_url'),
    sqlc.narg('fork'),
    sqlc.narg('visibility'),
    sqlc.narg('default_branch'),
    COALESCE(sqlc.narg('archived'), FALSE),
    sqlc.narg('disabled'),
    sqlc.narg('pushed_at'),
    sqlc.narg('created_at'),
    sqlc.narg('updated_at'),
    sqlc.narg('raw')
)
ON CONFLICT (full_name) DO UPDATE
SET github_id = EXCLUDED.github_id,
    node_id = EXCLUDED.node_id,
    name = EXCLUDED.name,
    owner_login = EXCLUDED.owner_login,
    owner_id = EXCLUDED.owner_id,
    private = EXCLUDED.private,
    description = EXCLUDED.description,
    html_url = EXCLUDED.html_url,
    owner_avatar_url = EXCLUDED.owner_avatar_url,
    owner_html_url = EXCLUDED.owner_html_url,
    fork = EXCLUDED.fork,
    visibility = EXCLUDED.visibility,
    default_branch = EXCLUDED.default_branch,
    archived = EXCLUDED.archived,
    disabled = EXCLUDED.disabled,
    pushed_at = EXCLUDED.pushed_at,
    created_at = COALESCE(EXCLUDED.created_at, repositories.created_at),
    updated_at = COALESCE(EXCLUDED.updated_at, repositories.updated_at),
    raw = EXCLUDED.raw
RETURNING *;

-- name: ListRepositories :many
SELECT *
FROM repositories
ORDER BY full_name;

-- name: GetRepositoryByID :one
SELECT *
FROM repositories
WHERE id = sqlc.arg('id');

-- name: FindRepositoryByFullName :one
SELECT *
FROM repositories
WHERE full_name = sqlc.arg('full_name');

