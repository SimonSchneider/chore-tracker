-- name: GetUser :one
SELECT *
FROM user
WHERE id = ?;

-- name: CreateUser :one
INSERT INTO user
    (id, created_at, updated_at)
VALUES (?, ?, ?) RETURNING *;

-- name: CreatePasswordAuth :exec
INSERT INTO password_auth
(user_id, username, hash)
VALUES (?, ?, ?);

-- name: GetPasswordAuthByUsername :one
SELECT *
FROM password_auth
WHERE username = ?;

-- name: GetPasswordAuthsByUser :many
SELECT username
FROM password_auth
WHERE user_id = ?;
