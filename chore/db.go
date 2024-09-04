package chore

import (
	"context"
	"database/sql"
	"time"
)

type Execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type Queryer interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}

type Beginner interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

func parseChoreWithEventRows(rows *sql.Rows, onChore func(chore *Chore)) error {
	var (
		tmpChore           Chore
		tmpEventID         *string
		tmpEventOccurredAt *time.Time
		chore              Chore
	)
	for rows.Next() {
		if err := rows.Scan(&tmpChore.ID, &tmpChore.Name, &tmpChore.Interval, &tmpEventID, &tmpEventOccurredAt); err != nil {
			return err
		}
		if chore.ID != tmpChore.ID {
			if chore.ID != "" {
				onChore(&chore)
			}
			chore = tmpChore
			chore.History = make([]Event, 0)
		}
		hist := chore.History
		lastCompl := chore.LastCompletion
		if tmpEventID != nil {
			hist = append(chore.History, Event{
				ID:         *tmpEventID,
				OccurredAt: *tmpEventOccurredAt,
			})
			lastCompl = maxTime(lastCompl, *tmpEventOccurredAt)
		}
		chore = tmpChore
		chore.History = hist
		chore.LastCompletion = lastCompl
	}
	if chore.ID != "" {
		onChore(&chore)
	}
	return nil
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}
