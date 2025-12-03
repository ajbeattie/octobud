-- name: ListRules :many
SELECT *
FROM rules
ORDER BY display_order ASC, id ASC;

-- name: ListEnabledRulesOrdered :many
SELECT *
FROM rules
WHERE enabled = TRUE
ORDER BY display_order ASC, id ASC;

-- name: GetRule :one
SELECT *
FROM rules
WHERE id = sqlc.arg('id');

-- name: CreateRule :one
INSERT INTO rules (
    name,
    description,
    query,
    view_id,
    enabled,
    actions,
    display_order
)
VALUES (
    sqlc.arg('name'),
    sqlc.narg('description'),
    sqlc.narg('query'),
    sqlc.narg('view_id'),
    sqlc.arg('enabled'),
    sqlc.arg('actions'),
    sqlc.arg('display_order')
)
RETURNING *;

-- name: UpdateRule :one
UPDATE rules
SET name = COALESCE(sqlc.narg('name'), name),
    description = COALESCE(sqlc.narg('description'), description),
    query = CASE
        WHEN sqlc.narg('clear_query')::boolean = true THEN NULL
        ELSE COALESCE(sqlc.narg('query'), query)
    END,
    view_id = CASE
        WHEN sqlc.narg('clear_view_id')::boolean = true THEN NULL
        ELSE COALESCE(sqlc.narg('view_id'), view_id)
    END,
    enabled = COALESCE(sqlc.narg('enabled'), enabled),
    actions = COALESCE(sqlc.narg('actions'), actions),
    updated_at = NOW()
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: UpdateRuleOrder :exec
UPDATE rules
SET display_order = sqlc.arg('display_order')
WHERE id = sqlc.arg('id');

-- name: DeleteRule :exec
DELETE FROM rules
WHERE id = sqlc.arg('id');

-- name: GetRulesByViewID :many
SELECT *
FROM rules
WHERE view_id = sqlc.arg('view_id');

