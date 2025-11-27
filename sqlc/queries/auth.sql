-- name: CreateToken :exec
INSERT INTO tokens
    (user_id, token, expires_at)
VALUES (?, ?, ?);

-- name: GetToken :one
SELECT *
FROM tokens
WHERE token = ?
  AND expires_at > ?;

-- name: DeleteTokensByUserId :exec
DELETE
FROM tokens
WHERE user_id = ?;

-- name: GetTokensByUser :many
SELECT *
FROM tokens
WHERE user_id = ?;