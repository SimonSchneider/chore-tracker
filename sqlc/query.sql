-- name: GetChore :one
SELECT * FROM chore WHERE id = ?;

-- name: GetChores :many
SELECT * FROM chore;
