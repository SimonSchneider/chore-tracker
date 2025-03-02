-- name: GetChore :one
SELECT *
FROM chore
WHERE id = ?;

-- name: ListChores :many
SELECT *
FROM chore
ORDER BY last_completion DESC, name ASC, id ASC;

-- name: CreateChore :one
INSERT INTO chore
    (id, name, interval, created_at, last_completion, snoozed_for)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateChore :exec
UPDATE chore
SET name     = ?,
    interval = ?
WHERE id = ?;

-- name: DeleteChore :exec
DELETE
FROM chore
WHERE id = ?;

-- name: CompleteChore :exec
UPDATE chore
SET last_completion = ?,
    snoozed_for     = 0
WHERE id = ?;

-- name: CreateChoreEvent :exec
INSERT INTO chore_event
    (id, chore_id, occurred_at)
VALUES (?, ?, ?);

-- name: SnoozeChore :exec
UPDATE chore
SET snoozed_for = ?
WHERE id = ?;

-- name: GetUser :one
SELECT *
FROM user
WHERE id = ?;

-- name: GetPasswordAuthByUsername :one
SELECT *
FROM password_auth
WHERE username = ?;

-- name: CreateUser :one
INSERT INTO user
    (id, created_at, updated_at)
VALUES (?, ?, ?)
RETURNING *;

-- name: CreateInvite :one
INSERT INTO invitation
    (id, created_at, expires_at, chore_list_id, created_by)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: GetInvite :one
SELECT inv.*, cl.name as chore_list_name, pa.username as created_by_name
FROM invitation inv
         LEFT JOIN chore_list cl ON inv.chore_list_id = cl.id
         LEFT JOIN user u on inv.created_by = u.id
         LEFT JOIN password_auth pa on u.id = pa.user_id
WHERE inv.id = ?
  AND inv.expires_at > ?;

-- name: AddUserToChoreList :exec
INSERT INTO chore_list_members
    (chore_list_id, user_id)
VALUES (?, ?);

-- name: DeleteInvite :one
DELETE
FROM invitation
WHERE id = ?
  AND expires_at > ?
RETURNING *;

-- name: CreatePasswordAuth :exec
INSERT INTO password_auth
    (user_id, username, hash)
VALUES (?, ?, ?);

-- name: ChoreListsForUser :many
SELECT cl.*
FROM chore_list cl
         JOIN chore_list_members clm ON cl.id = clm.chore_list_id
WHERE clm.user_id = ?;

-- name: GetChoreListByUser :one
SELECT cl.*
FROM chore_list cl
         JOIN chore_list_members clm ON cl.id = clm.chore_list_id
WHERE clm.user_id = ?
  AND cl.id = ?;

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
