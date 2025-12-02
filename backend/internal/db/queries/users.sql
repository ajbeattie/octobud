-- name: GetUser :one
SELECT * FROM users
LIMIT 1;

-- name: CreateUser :one
INSERT INTO users (
    username,
    password_hash
)
VALUES (
    sqlc.arg('username'),
    sqlc.arg('password_hash')
)
RETURNING *;

-- name: UpdateUserUsername :one
UPDATE users
SET username = sqlc.arg('username'),
    updated_at = NOW()
RETURNING *;

-- name: UpdateUserPassword :one
UPDATE users
SET password_hash = sqlc.arg('password_hash'),
    updated_at = NOW()
RETURNING *;

-- name: UpdateUserCredentials :one
UPDATE users
SET username = sqlc.arg('username'),
    password_hash = sqlc.arg('password_hash'),
    updated_at = NOW()
RETURNING *;

-- name: UpdateUserSyncSettings :one
UPDATE users
SET sync_settings = sqlc.arg('sync_settings'),
    updated_at = NOW()
RETURNING *;
