package chore

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"net/http"
	"time"
)

type Chore struct {
	ID             string    `json:"id,omitempty"`
	Name           string    `json:"name,omitempty"`
	Interval       Duration  `json:"interval,omitempty"`
	LastCompletion time.Time `json:"last_completion,omitempty"`
	History        []Event   `json:"history,omitempty"`
}

type Event struct {
	ID         string    `json:"id,omitempty"`
	OccurredAt time.Time `json:"occurred_at,omitempty"`
}

func Setup(ctx context.Context, db Execer) error {
	_, err := db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS chore (id TEXT NOT NULL PRIMARY KEY, name TEXT NOT NULL, interval INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS chore_event (id TEXT NOT NULL PRIMARY KEY, chore_id TEXT NOT NULL, occurred_at DATETIME NOT NULL, FOREIGN KEY (chore_id) REFERENCES chore(id));
`)
	if err != nil {
		return fmt.Errorf("execing setup query: %w", err)
	}
	return nil
}

func List(ctx context.Context, db Queryer) ([]Chore, error) {
	rows, err := db.QueryContext(ctx, "SELECT c.id, c.name, c.interval, e.id, e.occurred_at FROM chore c LEFT OUTER JOIN chore_event e ON c.id = e.chore_id ORDER BY e.occurred_at DESC")
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

type Input struct {
	Name     string   `json:"name"`
	Interval Duration `json:"interval"`
}

func (i *Input) FromForm(r *http.Request) error {
	i.Name = r.FormValue("name")
	interVal := r.FormValue("interval")
	inter, err := ParseDuration(interVal)
	if err != nil {
		return fmt.Errorf("illegal interval '%s': %w", interVal, err)
	}
	i.Interval = inter
	return nil
}

func Create(ctx context.Context, db Preparer, input Input) (*Chore, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("illegal empty name for new chore")
	}
	if input.Interval < Duration(time.Hour) {
		return nil, fmt.Errorf("chore interval can't be shorter than one hour")
	}
	p, err := db.PrepareContext(ctx, "INSERT INTO chore (id, name, interval) VALUES (?, ?, ?)")
	if err != nil {
		return nil, err
	}
	defer p.Close()
	chore := Chore{
		ID:       uuid.NewString(),
		Name:     input.Name,
		Interval: input.Interval,
	}
	_, err = p.ExecContext(ctx, chore.ID, input.Name, input.Interval)
	if err != nil {
		return nil, fmt.Errorf("scanning id into chore: %w", err)
	}
	return &chore, nil
}

func Get(ctx context.Context, db Queryer, id string) (*Chore, error) {
	rows, err := db.QueryContext(ctx, "SELECT c.id, c.name, c.interval, e.id, e.occurred_at FROM chore c LEFT OUTER JOIN chore_event e ON c.id = e.chore_id WHERE c.id = $1 ORDER BY e.occurred_at DESC", id)
	if err != nil {
		return nil, fmt.Errorf("finding chore %s: %w", id, err)
	}
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

func CompleteNow(ctx context.Context, db interface {
	Preparer
	Queryer
}, id string) (*Chore, error) {
	return Complete(ctx, db, id, time.Now())
}

func Complete(ctx context.Context, db interface {
	Preparer
	Queryer
}, id string, occurredAt time.Time) (*Chore, error) {
	p, err := db.PrepareContext(ctx, "INSERT INTO chore_event(id, chore_id, occurred_at) VALUES(?, ?, ?)")
	if err != nil {
		return nil, fmt.Errorf("cannot prepare: %w", err)
	}
	defer p.Close()
	if occurredAt.IsZero() {
		occurredAt = time.Now()
	}
	event := Event{
		ID:         uuid.NewString(),
		OccurredAt: occurredAt,
	}
	if _, err := p.ExecContext(ctx, event.ID, id, event.OccurredAt); err != nil {
		return nil, fmt.Errorf("inserting new event: %w", err)
	}
	return Get(ctx, db, id)
}

func DeleteCompletion(ctx context.Context, db Preparer, choreId, eventId string) error {
	p, err := db.PrepareContext(ctx, "DELETE FROM chore_event WHERE chore_id = ? AND id = ?")
	if err != nil {
		return fmt.Errorf("preparing delete: %w", err)
	}
	defer p.Close()
	if _, err := p.ExecContext(ctx, choreId, eventId); err != nil {
		return fmt.Errorf("deleting completion: %w", err)
	}
	return nil
}
