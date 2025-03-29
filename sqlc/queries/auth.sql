-- name: CreateToken :exec
INSERT INTO tokens
    (user_id, token, csrf_token, expires_at)
VALUES (?, ?, ?, ?);

-- name: GetToken :one
SELECT *
FROM tokens
WHERE token = ?
  AND expires_at > ?;

-- name: DeleteToken :exec
DELETE
FROM tokens
WHERE token = ?;

-- name: DeleteTokensByUserId :exec
DELETE
FROM tokens
WHERE user_id = ?;

-- name: GetTokensByUser :many
SELECT *
FROM tokens
WHERE user_id = ?;

-- name: GetCsrfTokenByUserAndCsrfToken :one
SELECT COUNT(*)
FROM tokens
WHERE user_id = ?
  AND csrf_token = ?
  AND expires_at > ?;

-- name: DeleteExpiredTokens :exec
DELETE
FROM tokens
WHERE expires_at < ?;