-- +goose Up
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Constraint to ensure only one user exists (single-user app)
CREATE UNIQUE INDEX IF NOT EXISTS users_single_user ON users((1));

-- +goose Down
DROP TABLE IF EXISTS users;

