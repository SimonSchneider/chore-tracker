package chore

import (
	"context"
	"fmt"
	"github.com/SimonSchneider/go-testing/duration"
	"github.com/google/uuid"
	"net/http"
	"time"
)

/*
TODO: use the events table as a log of what happened
(completed, snoozed) but also write it to the chore table
to make it easier to query the next completion date.
use the events to undo etc.
*/
func Setup(ctx context.Context, db Execer) error {
	_, err := db.ExecContext(ctx, `
PRAGMA foreign_keys = ON;
CREATE TABLE IF NOT EXISTS chore (id TEXT NOT NULL PRIMARY KEY, name TEXT NOT NULL, interval INTEGER NOT NULL, last_completion DATETIME);
CREATE TABLE IF NOT EXISTS chore_event (id TEXT NOT NULL PRIMARY KEY, chore_id TEXT NOT NULL, occurred_at DATETIME NOT NULL, FOREIGN KEY (chore_id) REFERENCES chore(id) ON DELETE CASCADE);
`)
	if err != nil {
		return fmt.Errorf("execing setup query: %w", err)
	}
	return nil
}

type Input struct {
	Name     string            `json:"name"`
	Interval duration.Duration `json:"interval"`
}

func (i *Input) FromForm(r *http.Request) error {
	i.Name = r.FormValue("name")
	interVal := r.FormValue("interval")
	inter, err := duration.ParseDuration(interVal)
	if err != nil {
		return fmt.Errorf("illegal interval '%s': %w", interVal, err)
	}
	i.Interval = inter
	return nil
}

func List(ctx context.Context, db Queryer) ([]Chore, error) {
	rows, err := db.QueryContext(ctx, "SELECT id, name, interval, clast_completion FROM chore ORDER BY occurred_at DESC, name ASC, id ASC")
	if err != nil {
		return nil, fmt.Errorf("querying chores: %w", err)
	}
	defer rows.Close()
	chores := make([]Chore, 0)
	if err := parseChoreWithEventRows(rows, func(chore *Chore) {
		chores = append(chores, *chore)
	}); err != nil {
		return nil, err
	}
	return chores, nil
}

func Get(ctx context.Context, db Queryer, id string) (*Chore, error) {
	rows, err := db.QueryContext(ctx, "SELECT c.id, c.name, c.interval, e.id, e.occurred_at FROM chore c LEFT OUTER JOIN chore_event e ON c.id = e.chore_id WHERE c.id = $1 ORDER BY e.occurred_at DESC", id)
	if err != nil {
		return nil, fmt.Errorf("finding chore %s: %w", id, err)
	}
	defer rows.Close()
	var chore *Chore
	if err := parseChoreWithEventRows(rows, func(newChore *Chore) {
		if chore != nil {
			err = fmt.Errorf("unexpected number of chores found")
		}
		chore = newChore
	}); err != nil {
		return nil, fmt.Errorf("parsing rows for chore %s: %w", id, err)
	}
	return chore, nil
}

func Create(ctx context.Context, db Beginner, input Input) (*Chore, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("illegal empty name for new chore")
	}
	if input.Interval.Zero() {
		return nil, fmt.Errorf("chore interval can't be zero")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("beginning tx: %w", err)
	}
	defer tx.Commit()
	chore := Chore{
		ID:       uuid.NewString(),
		Name:     input.Name,
		Interval: input.Interval,
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO chore (id, name, interval) VALUES (?, ?, ?)", chore.ID, input.Name, input.Interval); err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("scanning id into chore: %w", err)
	}
	return &chore, nil
}

func Update(ctx context.Context, db Beginner, id string, input Input) error {
	if id == "" {
		return fmt.Errorf("illegal empty id for updating chore")
	}
	if input.Name == "" || input.Interval.Zero() {
		return fmt.Errorf("illegal empty name or interval for updating chore")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning tx: %w", err)
	}
	defer tx.Commit()
	if _, err := tx.ExecContext(ctx, "UPDATE chore SET name = ?, interval = ? WHERE id = ?", input.Name, input.Interval, id); err != nil {
		tx.Rollback()
		return fmt.Errorf("updating chore: %w", err)
	}
	return nil
}

func Delete(ctx context.Context, db Beginner, id string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning tx: %w", err)
	}
	defer tx.Commit()
	if _, err := tx.ExecContext(ctx, "DELETE FROM chore WHERE id = ?", id); err != nil {
		tx.Rollback()
		return fmt.Errorf("deleting chore: %w", err)
	}
	return nil
}

func Complete(ctx context.Context, db Beginner, id string, occurredAt time.Time) error {
	// TODO: don't complete if already completed on this day.
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning tx: %w", err)
	}
	defer tx.Commit()
	if occurredAt.IsZero() {
		occurredAt = time.Now()
	}
	event := Event{
		ID:         uuid.NewString(),
		OccurredAt: occurredAt,
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO chore_event(id, chore_id, occurred_at) VALUES(?, ?, ?)", event.ID, id, event.OccurredAt); err != nil {
		tx.Rollback()
		return fmt.Errorf("inserting new event: %w", err)
	}
	return nil
}

func DeleteCompletion(ctx context.Context, db Beginner, choreId, eventId string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning tx: %w", err)
	}
	defer tx.Commit()
	if _, err := tx.ExecContext(ctx, "DELETE FROM chore_event WHERE chore_id = ? AND id = ?", choreId, eventId); err != nil {
		tx.Rollback()
		return fmt.Errorf("deleting completion: %w", err)
	}
	return nil
}
