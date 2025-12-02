-- name: CreateView :one
INSERT INTO views (
    name,
    slug,
    description,
    icon,
    is_default,
    query
)
VALUES (
    sqlc.arg('name'),
    sqlc.arg('slug'),
    sqlc.narg('description'),
    sqlc.narg('icon'),
    COALESCE(sqlc.narg('is_default'), FALSE),
    sqlc.narg('query')
)
RETURNING *;

-- name: UpdateView :one
UPDATE views
SET name = COALESCE(sqlc.narg('name'), views.name),
    slug = COALESCE(sqlc.narg('slug'), views.slug),
    description = COALESCE(sqlc.narg('description'), views.description),
    icon = COALESCE(sqlc.narg('icon'), views.icon),
    is_default = COALESCE(sqlc.narg('is_default'), views.is_default),
    query = COALESCE(sqlc.narg('query'), views.query)
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: DeleteView :one
DELETE FROM views
WHERE id = sqlc.arg('id')
RETURNING id;

-- name: ListViews :many
SELECT *
FROM views
ORDER BY 
    CASE 
        WHEN display_order = 0 THEN name 
        ELSE LPAD(display_order::text, 10, '0') 
    END,
    name;

-- name: GetView :one
SELECT *
FROM views
WHERE id = sqlc.arg('id')
LIMIT 1;

-- name: GetViewBySlug :one
SELECT *
FROM views
WHERE slug = sqlc.arg('slug')
LIMIT 1;

-- name: UpdateViewOrder :exec
UPDATE views
SET display_order = sqlc.arg('display_order')
WHERE id = sqlc.arg('id');
