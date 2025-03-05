// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.28.0
// source: query.sql

package cdb

import (
	"context"
)

const addUserToChoreList = `-- name: AddUserToChoreList :exec
INSERT INTO chore_list_members
    (chore_list_id, user_id)
VALUES (?, ?)
`

type AddUserToChoreListParams struct {
	ChoreListID string
	UserID      string
}

func (q *Queries) AddUserToChoreList(ctx context.Context, arg AddUserToChoreListParams) error {
	_, err := q.db.ExecContext(ctx, addUserToChoreList, arg.ChoreListID, arg.UserID)
	return err
}

const completeChore = `-- name: CompleteChore :exec
UPDATE chore
SET last_completion = ?,
    snoozed_for     = 0
WHERE id = ?
`

type CompleteChoreParams struct {
	LastCompletion int64
	ID             string
}

func (q *Queries) CompleteChore(ctx context.Context, arg CompleteChoreParams) error {
	_, err := q.db.ExecContext(ctx, completeChore, arg.LastCompletion, arg.ID)
	return err
}

const createChore = `-- name: CreateChore :one
INSERT INTO chore
    (id, name, interval, created_at, last_completion, snoozed_for, chore_list_id, created_by)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id, name, interval, last_completion, snoozed_for, created_at, chore_list_id, created_by
`

type CreateChoreParams struct {
	ID             string
	Name           string
	Interval       int64
	CreatedAt      int64
	LastCompletion int64
	SnoozedFor     int64
	ChoreListID    string
	CreatedBy      string
}

func (q *Queries) CreateChore(ctx context.Context, arg CreateChoreParams) (Chore, error) {
	row := q.db.QueryRowContext(ctx, createChore,
		arg.ID,
		arg.Name,
		arg.Interval,
		arg.CreatedAt,
		arg.LastCompletion,
		arg.SnoozedFor,
		arg.ChoreListID,
		arg.CreatedBy,
	)
	var i Chore
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Interval,
		&i.LastCompletion,
		&i.SnoozedFor,
		&i.CreatedAt,
		&i.ChoreListID,
		&i.CreatedBy,
	)
	return i, err
}

const createChoreEvent = `-- name: CreateChoreEvent :exec
INSERT INTO chore_event
    (id, chore_id, event_type, created_by, occurred_at)
VALUES (?, ?, ?, ?, ?)
`

type CreateChoreEventParams struct {
	ID         string
	ChoreID    string
	EventType  string
	CreatedBy  string
	OccurredAt int64
}

func (q *Queries) CreateChoreEvent(ctx context.Context, arg CreateChoreEventParams) error {
	_, err := q.db.ExecContext(ctx, createChoreEvent,
		arg.ID,
		arg.ChoreID,
		arg.EventType,
		arg.CreatedBy,
		arg.OccurredAt,
	)
	return err
}

const createChoreList = `-- name: CreateChoreList :one
INSERT INTO chore_list
    (id, name, created_at, updated_at)
VALUES (?, ?, ?, ?)
RETURNING id, created_at, updated_at, name
`

type CreateChoreListParams struct {
	ID        string
	Name      string
	CreatedAt int64
	UpdatedAt int64
}

func (q *Queries) CreateChoreList(ctx context.Context, arg CreateChoreListParams) (ChoreList, error) {
	row := q.db.QueryRowContext(ctx, createChoreList,
		arg.ID,
		arg.Name,
		arg.CreatedAt,
		arg.UpdatedAt,
	)
	var i ChoreList
	err := row.Scan(
		&i.ID,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.Name,
	)
	return i, err
}

const deleteChore = `-- name: DeleteChore :exec
DELETE
FROM chore
WHERE id = ?
`

func (q *Queries) DeleteChore(ctx context.Context, id string) error {
	_, err := q.db.ExecContext(ctx, deleteChore, id)
	return err
}

const getChore = `-- name: GetChore :one
SELECT id, name, interval, last_completion, snoozed_for, created_at, chore_list_id, created_by
FROM chore
WHERE id = ?
`

func (q *Queries) GetChore(ctx context.Context, id string) (Chore, error) {
	row := q.db.QueryRowContext(ctx, getChore, id)
	var i Chore
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Interval,
		&i.LastCompletion,
		&i.SnoozedFor,
		&i.CreatedAt,
		&i.ChoreListID,
		&i.CreatedBy,
	)
	return i, err
}

const getChoreListByUser = `-- name: GetChoreListByUser :one
SELECT cl.id, cl.created_at, cl.updated_at, cl.name
FROM chore_list cl
         JOIN chore_list_members clm ON cl.id = clm.chore_list_id
WHERE clm.user_id = ?
  AND cl.id = ?
`

type GetChoreListByUserParams struct {
	UserID string
	ID     string
}

func (q *Queries) GetChoreListByUser(ctx context.Context, arg GetChoreListByUserParams) (ChoreList, error) {
	row := q.db.QueryRowContext(ctx, getChoreListByUser, arg.UserID, arg.ID)
	var i ChoreList
	err := row.Scan(
		&i.ID,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.Name,
	)
	return i, err
}

const getChoreListsByUser = `-- name: GetChoreListsByUser :many
SELECT cl.id, cl.created_at, cl.updated_at, cl.name
FROM chore_list cl
         JOIN chore_list_members clm ON cl.id = clm.chore_list_id
WHERE clm.user_id = ?
`

func (q *Queries) GetChoreListsByUser(ctx context.Context, userID string) ([]ChoreList, error) {
	rows, err := q.db.QueryContext(ctx, getChoreListsByUser, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ChoreList
	for rows.Next() {
		var i ChoreList
		if err := rows.Scan(
			&i.ID,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.Name,
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

const listChores = `-- name: ListChores :many
SELECT id, name, interval, last_completion, snoozed_for, created_at, chore_list_id, created_by
FROM chore
ORDER BY last_completion DESC, name, id
`

func (q *Queries) ListChores(ctx context.Context) ([]Chore, error) {
	rows, err := q.db.QueryContext(ctx, listChores)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Chore
	for rows.Next() {
		var i Chore
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.Interval,
			&i.LastCompletion,
			&i.SnoozedFor,
			&i.CreatedAt,
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

const snoozeChore = `-- name: SnoozeChore :exec
UPDATE chore
SET snoozed_for = ?
WHERE id = ?
`

type SnoozeChoreParams struct {
	SnoozedFor int64
	ID         string
}

func (q *Queries) SnoozeChore(ctx context.Context, arg SnoozeChoreParams) error {
	_, err := q.db.ExecContext(ctx, snoozeChore, arg.SnoozedFor, arg.ID)
	return err
}

const updateChore = `-- name: UpdateChore :exec
UPDATE chore
SET name     = ?,
    interval = ?
WHERE id = ?
`

type UpdateChoreParams struct {
	Name     string
	Interval int64
	ID       string
}

func (q *Queries) UpdateChore(ctx context.Context, arg UpdateChoreParams) error {
	_, err := q.db.ExecContext(ctx, updateChore, arg.Name, arg.Interval, arg.ID)
	return err
}
