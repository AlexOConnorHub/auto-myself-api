
-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: CreateUser :exec
INSERT INTO users (id, name) VALUES ($1, $2);
