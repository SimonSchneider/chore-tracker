-- name: GetChore :one
SELECT chore.*
FROM chore
         JOIN chore_list cl ON chore.chore_list_id = cl.id
         JOIN chore_list_members ON cl.id = chore_list_members.chore_list_id
WHERE chore.id = ?
  AND chore_list_members.user_id = ?;

-- name: CreateChore :one
INSERT INTO chore
(id, name, interval, created_at, last_completion, snoozed_for, chore_list_id, created_by)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateChore :one
UPDATE chore
SET name     = ?,
    interval = ?
WHERE id = ?
RETURNING *;

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
    (id, chore_id, event_type, created_by, occurred_at)
VALUES (?, ?, ?, ?, ?);

-- name: SnoozeChore :exec
UPDATE chore
SET snoozed_for = ?
WHERE id = ?;

-- name: GetChoresByList :many
SELECT *
FROM chore
WHERE chore_list_id = ?
ORDER BY last_completion DESC, name, id;

-- name: GetChoreListByUser :one
SELECT cl.*
FROM chore_list cl
         JOIN chore_list_members clm ON cl.id = clm.chore_list_id
WHERE clm.user_id = ?
  AND cl.id = ?;

-- name: GetChoreListsByUser :many
SELECT cl.*
FROM chore_list cl
         JOIN chore_list_members clm ON cl.id = clm.chore_list_id
WHERE clm.user_id = ?;

-- name: CreateChoreList :one
INSERT INTO chore_list
    (id, name, created_at, updated_at)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: AddUserToChoreList :exec
INSERT INTO chore_list_members
    (chore_list_id, user_id)
VALUES (?, ?);
