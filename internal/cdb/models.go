// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0

package cdb

type Chore struct {
	ID             string
	Name           string
	Interval       int64
	LastCompletion int64
	SnoozedFor     int64
	CreatedAt      int64
}

type ChoreEvent struct {
	ID         string
	ChoreID    string
	OccurredAt int64
}
