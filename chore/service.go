package chore

import (
	"context"
	"fmt"
	"github.com/SimonSchneider/go-testing/date"
	"github.com/SimonSchneider/go-testing/srvu"
	"github.com/google/uuid"
	"net/http"
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
CREATE TABLE IF NOT EXISTS chore (id TEXT NOT NULL PRIMARY KEY, name TEXT NOT NULL, interval INTEGER NOT NULL, last_completion INTEGER NOT NULL DEFAULT 0, snoozed_for INTEGER NOT NULL DEFAULT 0);
CREATE TABLE IF NOT EXISTS chore_event (id TEXT NOT NULL PRIMARY KEY, chore_id TEXT NOT NULL, occurred_at INTEGER NOT NULL, FOREIGN KEY (chore_id) REFERENCES chore(id) ON DELETE CASCADE);
`)
	if err != nil {
		return fmt.Errorf("execing setup query: %w", err)
	}
	return nil
}

type Input struct {
	Name     string        `json:"name"`
	Interval date.Duration `json:"interval"`
}

func (i *Input) FromForm(r *http.Request) error {
	i.Name = r.FormValue("name")
	interVal := r.FormValue("interval")
	inter, err := date.ParseDuration(interVal)
	if err != nil {
		return fmt.Errorf("illegal interval '%s': %w", interVal, err)
	}
	i.Interval = inter
	return nil
}

func List(ctx context.Context, db Queryer) ([]Chore, error) {
	rows, err := db.QueryContext(ctx, "SELECT id, name, interval, last_completion, snoozed_for FROM chore ORDER BY last_completion DESC, name ASC, id ASC")
	if err != nil {
		return nil, fmt.Errorf("querying chores: %w", err)
	}
	return parseChoreRows(rows)
}

func Get(ctx context.Context, db RowQueryer, id string) (*Chore, error) {
	row := db.QueryRowContext(ctx, "SELECT id, name, interval, last_completion, snoozed_for FROM chore WHERE id = ?", id)
	var chore Chore
	if err := parseChoreRow(row, &chore); err != nil {
		return nil, fmt.Errorf("querying chore %s: %w", id, err)
	}
	return &chore, nil
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

func Complete(ctx context.Context, db Beginner, id string, occurredAt date.Date) error {
	// TODO: don't complete if already completed on this day.
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning tx: %w", err)
	}
	defer tx.Commit()
	if occurredAt.IsZero() {
		occurredAt = date.Today()
	}
	event := Event{
		ID:         uuid.NewString(),
		OccurredAt: occurredAt,
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO chore_event(id, chore_id, occurred_at) VALUES(?, ?, ?)", event.ID, id, event.OccurredAt); err != nil {
		tx.Rollback()
		return fmt.Errorf("inserting new event: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "UPDATE chore SET last_completion = ?, snoozed_for = 0 WHERE id = ?", event.OccurredAt, id); err != nil {
		tx.Rollback()
		return fmt.Errorf("updating last completion: %w", err)
	}
	return nil
}

func Snooze(ctx context.Context, db Beginner, id string, snoozeFor date.Duration) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning tx: %w", err)
	}
	defer tx.Commit()
	ex, err := Get(ctx, tx, id)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("getting chore: %w", err)
	}
	durToNext := ex.DurationToNext()
	if durToNext > 0 {
		return fmt.Errorf("can't snooze a chore that is not due: %s", durToNext)
	}
	snoozedFor := durToNext + snoozeFor
	srvu.GetLogger(ctx).Printf("(%s - %s)+%s=%s", ex.NextCompletion(), date.Today(), snoozeFor, snoozedFor)
	if snoozedFor < 0 {
		if _, err := tx.ExecContext(ctx, "UPDATE chore SET snoozed_for = 0 WHERE id = ?", id); err != nil {
			tx.Rollback()
			return fmt.Errorf("resetting snooze: %w", err)
		}
	} else {
		if _, err := tx.ExecContext(ctx, "UPDATE chore SET snoozed_for = ? WHERE id = ?", snoozedFor, id); err != nil {
			tx.Rollback()
			return fmt.Errorf("updating snooze duration: %w", err)
		}
	}
	return nil
}
