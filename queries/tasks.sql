-- name: InsertUserTask :one
INSERT INTO users.tasks (
    source,
    original_input
) VALUES (
    $1,
    $2
)
RETURNING task_id;

-- name: GetUserTask :one
SELECT * FROM users.tasks
WHERE task_id = $1;

-- name: UpdateUserTaskStatus :exec
UPDATE users.tasks
SET status = sqlc.arg('task_status')::task_status, updated_at = NOW()
WHERE task_id = $1;

-- name: UpdateUserTaskErrMsg :exec
UPDATE users.tasks
SET error_message = $1, status = 'failed', updated_at = NOW()
WHERE task_id = $2;

-- name: ListUserTasks :many
SELECT * FROM users.tasks
WHERE id > $1
ORDER BY id DESC
LIMIT sqlc.arg('limit')::integer;

