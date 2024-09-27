package chore

import (
	"context"
	"database/sql"
	"fmt"
)

type Execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type Queryer interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}

type RowQueryer interface {
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

type Beginner interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

func parseChoreRow(row interface{ Scan(args ...any) error }, chore *Chore) error {
	if err := row.Scan(&chore.ID, &chore.Name, &chore.Interval, &chore.LastCompletion, &chore.SnoozedFor); err != nil {
		return fmt.Errorf("scanning chore: %w", err)
	}
	return nil
}

func parseChoreRows(rows *sql.Rows) ([]Chore, error) {
	defer rows.Close()
	chores := make([]Chore, 0)
	for rows.Next() {
		var chore Chore
		if err := parseChoreRow(rows, &chore); err != nil {
			return nil, err
		}
		chores = append(chores, chore)
	}
	return chores, nil
}
