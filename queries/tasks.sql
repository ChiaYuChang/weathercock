-- name: CreateTask :one
INSERT INTO users.tasks (
    source,
    original_input
) VALUES (
    $1,
    $2
)
RETURNING task_id;