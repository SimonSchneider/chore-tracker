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
    (id, name, interval, date_glob, created_at, last_completion, snoozed_for)
VALUES (?, ?, ?, ?, ?, ?, ?) RETURNING *;

-- name: UpdateChore :exec
UPDATE chore
SET name     = ?,
    interval = ?,
    date_glob = ?
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