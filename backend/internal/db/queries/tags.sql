-- name: UpsertTag :one
INSERT INTO tags (
    name,
    slug,
    color,
    description
)
VALUES (
    sqlc.arg('name'),
    sqlc.arg('slug'),
    sqlc.narg('color'),
    sqlc.narg('description')
)
ON CONFLICT (name) DO UPDATE
SET slug = EXCLUDED.slug,
    color = EXCLUDED.color,
    description = EXCLUDED.description
RETURNING *;

-- name: GetTag :one
SELECT * FROM tags
WHERE id = sqlc.arg('id')
LIMIT 1;

-- name: UpdateTag :one
UPDATE tags
SET name = sqlc.arg('name'),
    slug = sqlc.arg('slug'),
    color = sqlc.narg('color'),
    description = sqlc.narg('description')
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: DeleteTag :exec
DELETE FROM tags
WHERE id = sqlc.arg('id');

-- name: AssignTagToEntity :one
INSERT INTO tag_assignments (
    tag_id,
    entity_type,
    entity_id
)
VALUES (
    sqlc.arg('tag_id'),
    sqlc.arg('entity_type'),
    sqlc.arg('entity_id')
)
ON CONFLICT (tag_id, entity_type, entity_id) DO UPDATE
SET entity_type = EXCLUDED.entity_type
RETURNING *;

-- name: UpdateNotificationTagIds :exec
UPDATE notifications
SET tag_ids = (
    SELECT COALESCE(array_agg(DISTINCT tag_id), '{}')
    FROM tag_assignments
    WHERE entity_type = 'notification' AND entity_id = sqlc.arg('notification_id')
)
WHERE id = sqlc.arg('notification_id');

-- name: RemoveTagAssignment :exec
DELETE FROM tag_assignments
WHERE tag_id = sqlc.arg('tag_id')
  AND entity_type = sqlc.arg('entity_type')
  AND entity_id = sqlc.arg('entity_id');

-- name: ListTagsForEntity :many
SELECT t.*
FROM tags t
JOIN tag_assignments ta ON ta.tag_id = t.id
WHERE ta.entity_type = sqlc.arg('entity_type')
  AND ta.entity_id = sqlc.arg('entity_id')
ORDER BY t.name;

-- name: ListAllTags :many
SELECT * FROM tags
ORDER BY display_order, name;

-- name: GetTagByName :one
SELECT * FROM tags
WHERE name = sqlc.arg('name')
LIMIT 1;

-- name: GetTagBySlug :one
SELECT * FROM tags
WHERE slug = sqlc.arg('slug')
LIMIT 1;

-- name: UpdateTagDisplayOrder :exec
UPDATE tags
SET display_order = sqlc.arg('display_order')
WHERE id = sqlc.arg('id');

