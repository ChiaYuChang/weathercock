-- name: InsertModel :one
INSERT INTO models (name)
VALUES (@name::text)
RETURNING id;
-- name: GetModelByName :one
SELECT id, name
FROM models
WHERE name = @name::text
LIMIT 1;
-- name: GetModelByID :one
SELECT id, name
FROM models
WHERE id = @id::integer
LIMIT 1;
-- name: DeleteModelByID :exec
DELETE FROM models
WHERE id = @id::integer
RETURNING id;
-- name: ListModels :many
SELECT id, name
FROM models
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;