-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, name, api_key)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5
)
RETURNING *;

-- name: GetUser :one
SELECT * FROM users WHERE name = $1;

-- name: GetUserByAPIKey :one
SELECT * FROM users WHERE api_key = $1;

-- name: UpdateUserAPIKey :exec
UPDATE users SET api_key = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2;

-- name: GetUsers :many
SELECT * FROM users;

-- name: DeleteAllUsers :exec
DELETE FROM users;
