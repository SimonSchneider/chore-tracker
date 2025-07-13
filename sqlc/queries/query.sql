-- name: GetChore :one
SELECT chore.*
FROM chore
         JOIN chore_list cl ON chore.chore_list_id = cl.id
         JOIN chore_list_members ON cl.id = chore_list_members.chore_list_id
WHERE chore.id = ?
  AND chore_list_members.user_id = ?;

-- name: CreateChore :one
INSERT INTO chore
(id, name, interval, created_at, last_completion, snoozed_for, repeats_left, chore_list_id, created_by, chore_type)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING *;

-- name: UpdateChore :one
UPDATE chore
SET name            = ?,
    interval        = ?,
    repeats_left    = ?,
    snoozed_for     = ?,
    last_completion = ?
WHERE id = ? RETURNING *;

-- name: DeleteChore :exec
DELETE
FROM chore
WHERE id = ?;

-- name: CompleteChore :exec
UPDATE chore
SET last_completion = ?,
    repeats_left    = max(-1, repeats_left - 1),
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
  AND NOT repeats_left = 0
ORDER BY last_completion DESC, name, id;

-- name: GetChoreListByUser :one
SELECT cl.*
FROM chore_list cl
         JOIN chore_list_members clm ON cl.id = clm.chore_list_id
WHERE clm.user_id = ?
  AND cl.id = ?;

-- name: GetChoreListWithoutUser :one
SELECT cl.*
FROM chore_list cl
WHERE cl.id = ?;

-- name: GetChoreListCalendarCompletionData :many
SELECT ce.occurred_at, COUNT(*) AS count
FROM chore_event ce
    JOIN chore c
ON ce.chore_id = c.id
    JOIN chore_list_members clm ON c.chore_list_id = clm.chore_list_id
WHERE clm.user_id = ?
  AND c.chore_list_id = ?
GROUP BY 1
ORDER BY 1;

-- name: GetChoreListMembers :many
SELECT u.id, u.display_name
FROM user u
         JOIN chore_list_members clm ON u.id = clm.user_id
WHERE clm.chore_list_id = ?;

-- name: GetChoreListsByUser :many
SELECT cl.*,
       (SELECT COUNT(*) FROM chore WHERE chore_list_id = cl.id AND NOT repeats_left = 0) AS chore_count,
       (SELECT COUNT(*) FROM chore_list_members WHERE chore_list_id = cl.id)             AS member_count
FROM chore_list cl
         JOIN chore_list_members clm ON cl.id = clm.chore_list_id
WHERE clm.user_id = ?
ORDER BY cl.name;

-- name: CreateChoreList :one
INSERT INTO chore_list
    (id, name, created_at, updated_at)
VALUES (?, ?, ?, ?) RETURNING *;

-- name: UpdateChoreList :one
UPDATE chore_list
SET name       = ?,
    updated_at = ?
WHERE id = ?
  AND id IN (SELECT chore_list_id FROM chore_list_members WHERE user_id = ?) RETURNING *;

-- name: RemoveUserFromChoreList :exec
DELETE
FROM chore_list_members
WHERE chore_list_id = ?
  AND user_id = ?;

-- name: AddUserToChoreList :exec
INSERT INTO chore_list_members
    (chore_list_id, user_id)
VALUES (?, ?);
