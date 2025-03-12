-- name: CreateInvite :one
INSERT INTO invitation
    (id, created_at, expires_at, chore_list_id, created_by)
VALUES (?, ?, ?, ?, ?) RETURNING *;

-- name: GetInvite :one
SELECT inv.*, cl.name as chore_list_name, pa.username as created_by_name
FROM invitation inv
         LEFT JOIN chore_list cl ON inv.chore_list_id = cl.id
         LEFT JOIN user u on inv.created_by = u.id
         LEFT JOIN password_auth pa on u.id = pa.user_id
WHERE inv.id = ?
  AND inv.expires_at > ?;

-- name: DeleteInvite :one
DELETE
FROM invitation
WHERE id = ?
  AND expires_at > ? RETURNING *;

-- name: GetInvitationsByCreator :many
SELECT inv.*, cl.name as chore_list_name
FROM invitation inv
         LEFT JOIN chore_list cl ON inv.chore_list_id = cl.id
WHERE inv.created_by = ?
  AND inv.expires_at > ?;

-- name: GetInvitationsByChoreList :many
SELECT *
FROM invitation
WHERE chore_list_id = ?
  AND expires_at > ?;
