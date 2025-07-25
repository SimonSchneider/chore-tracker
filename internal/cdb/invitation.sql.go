// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.29.0
// source: invitation.sql

package cdb

import (
	"context"
	"database/sql"
)

const createInvite = `-- name: CreateInvite :one
INSERT INTO invitation
    (id, created_at, expires_at, chore_list_id, created_by)
VALUES (?, ?, ?, ?, ?) RETURNING id, created_at, expires_at, chore_list_id, created_by
`

type CreateInviteParams struct {
	ID          string
	CreatedAt   int64
	ExpiresAt   int64
	ChoreListID sql.NullString
	CreatedBy   string
}

func (q *Queries) CreateInvite(ctx context.Context, arg CreateInviteParams) (Invitation, error) {
	row := q.db.QueryRowContext(ctx, createInvite,
		arg.ID,
		arg.CreatedAt,
		arg.ExpiresAt,
		arg.ChoreListID,
		arg.CreatedBy,
	)
	var i Invitation
	err := row.Scan(
		&i.ID,
		&i.CreatedAt,
		&i.ExpiresAt,
		&i.ChoreListID,
		&i.CreatedBy,
	)
	return i, err
}

const deleteInvite = `-- name: DeleteInvite :one
DELETE
FROM invitation
WHERE id = ?
  AND expires_at > ? RETURNING id, created_at, expires_at, chore_list_id, created_by
`

type DeleteInviteParams struct {
	ID        string
	ExpiresAt int64
}

func (q *Queries) DeleteInvite(ctx context.Context, arg DeleteInviteParams) (Invitation, error) {
	row := q.db.QueryRowContext(ctx, deleteInvite, arg.ID, arg.ExpiresAt)
	var i Invitation
	err := row.Scan(
		&i.ID,
		&i.CreatedAt,
		&i.ExpiresAt,
		&i.ChoreListID,
		&i.CreatedBy,
	)
	return i, err
}

const deleteInviteByChoreList = `-- name: DeleteInviteByChoreList :exec
DELETE
FROM invitation
WHERE chore_list_id = ?
  AND id = ?
`

type DeleteInviteByChoreListParams struct {
	ChoreListID sql.NullString
	ID          string
}

func (q *Queries) DeleteInviteByChoreList(ctx context.Context, arg DeleteInviteByChoreListParams) error {
	_, err := q.db.ExecContext(ctx, deleteInviteByChoreList, arg.ChoreListID, arg.ID)
	return err
}

const getInvitationsByChoreList = `-- name: GetInvitationsByChoreList :many
SELECT id, created_at, expires_at, chore_list_id, created_by
FROM invitation
WHERE chore_list_id = ?
  AND expires_at > ?
`

type GetInvitationsByChoreListParams struct {
	ChoreListID sql.NullString
	ExpiresAt   int64
}

func (q *Queries) GetInvitationsByChoreList(ctx context.Context, arg GetInvitationsByChoreListParams) ([]Invitation, error) {
	rows, err := q.db.QueryContext(ctx, getInvitationsByChoreList, arg.ChoreListID, arg.ExpiresAt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Invitation
	for rows.Next() {
		var i Invitation
		if err := rows.Scan(
			&i.ID,
			&i.CreatedAt,
			&i.ExpiresAt,
			&i.ChoreListID,
			&i.CreatedBy,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getInvitationsByCreator = `-- name: GetInvitationsByCreator :many
SELECT inv.id, inv.created_at, inv.expires_at, inv.chore_list_id, inv.created_by, cl.name as chore_list_name
FROM invitation inv
         LEFT JOIN chore_list cl ON inv.chore_list_id = cl.id
WHERE inv.created_by = ?
  AND inv.expires_at > ?
`

type GetInvitationsByCreatorParams struct {
	CreatedBy string
	ExpiresAt int64
}

type GetInvitationsByCreatorRow struct {
	ID            string
	CreatedAt     int64
	ExpiresAt     int64
	ChoreListID   sql.NullString
	CreatedBy     string
	ChoreListName sql.NullString
}

func (q *Queries) GetInvitationsByCreator(ctx context.Context, arg GetInvitationsByCreatorParams) ([]GetInvitationsByCreatorRow, error) {
	rows, err := q.db.QueryContext(ctx, getInvitationsByCreator, arg.CreatedBy, arg.ExpiresAt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetInvitationsByCreatorRow
	for rows.Next() {
		var i GetInvitationsByCreatorRow
		if err := rows.Scan(
			&i.ID,
			&i.CreatedAt,
			&i.ExpiresAt,
			&i.ChoreListID,
			&i.CreatedBy,
			&i.ChoreListName,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getInvite = `-- name: GetInvite :one
SELECT inv.id, inv.created_at, inv.expires_at, inv.chore_list_id, inv.created_by, cl.name as chore_list_name, pa.username as created_by_name
FROM invitation inv
         LEFT JOIN chore_list cl ON inv.chore_list_id = cl.id
         LEFT JOIN user u on inv.created_by = u.id
         LEFT JOIN password_auth pa on u.id = pa.user_id
WHERE inv.id = ?
  AND inv.expires_at > ?
`

type GetInviteParams struct {
	ID        string
	ExpiresAt int64
}

type GetInviteRow struct {
	ID            string
	CreatedAt     int64
	ExpiresAt     int64
	ChoreListID   sql.NullString
	CreatedBy     string
	ChoreListName sql.NullString
	CreatedByName sql.NullString
}

func (q *Queries) GetInvite(ctx context.Context, arg GetInviteParams) (GetInviteRow, error) {
	row := q.db.QueryRowContext(ctx, getInvite, arg.ID, arg.ExpiresAt)
	var i GetInviteRow
	err := row.Scan(
		&i.ID,
		&i.CreatedAt,
		&i.ExpiresAt,
		&i.ChoreListID,
		&i.CreatedBy,
		&i.ChoreListName,
		&i.CreatedByName,
	)
	return i, err
}
