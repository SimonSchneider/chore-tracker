package chore

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/SimonSchneider/go-testing/internal/cdb"
	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/goslu/sid"
	"github.com/SimonSchneider/goslu/srvu"
	"net/http"
)

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

func List(ctx context.Context, db *sql.DB) ([]Chore, error) {
	dbChores, err := cdb.New(db).ListChores(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing chores: %w", err)
	}
	chores := make([]Chore, len(dbChores))
	for i, dbChore := range dbChores {
		chores[i] = ChoreFromDb(dbChore)
	}
	return chores, nil
}

func Get(ctx context.Context, db cdb.DBTX, id string) (*Chore, error) {
	row, err := cdb.New(db).GetChore(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("querying chore %s: %w", id, err)
	}
	chore := ChoreFromDb(row)
	return &chore, nil
}

func Create(ctx context.Context, db *sql.DB, input Input) (*Chore, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("illegal empty name for new chore")
	}
	if input.Interval.Zero() {
		return nil, fmt.Errorf("chore interval can't be zero")
	}
	row, err := cdb.New(db).CreateChore(ctx, cdb.CreateChoreParams{
		ID:       sid.MustNewString(15),
		Name:     input.Name,
		Interval: int64(input.Interval),
	})
	if err != nil {
		return nil, fmt.Errorf("creating chore: %w", err)
	}
	chore := ChoreFromDb(row)
	return &chore, nil
}

func Update(ctx context.Context, db *sql.DB, id string, input Input) error {
	if id == "" {
		return fmt.Errorf("illegal empty id for updating chore")
	}
	if input.Name == "" || input.Interval.Zero() {
		return fmt.Errorf("illegal empty name or interval for updating chore")
	}
	err := cdb.New(db).UpdateChore(ctx, cdb.UpdateChoreParams{
		ID:       id,
		Name:     input.Name,
		Interval: int64(input.Interval),
	})
	if err != nil {
		return fmt.Errorf("updating chore: %w", err)
	}
	return nil
}

func Delete(ctx context.Context, db *sql.DB, id string) error {
	if err := cdb.New(db).DeleteChore(ctx, id); err != nil {
		return fmt.Errorf("deleting chore: %w", err)
	}
	return nil
}

func Complete(ctx context.Context, db *sql.DB, id string, occurredAt date.Date) error {
	// TODO: don't complete if already completed on this day.
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning tx: %w", err)
	}
	defer tx.Commit()
	if occurredAt.IsZero() {
		occurredAt = date.Today()
	}
	occAt := int64(occurredAt)
	txc := cdb.New(tx)
	if err := txc.CreateChoreEvent(ctx, cdb.CreateChoreEventParams{ID: sid.MustNewString(15), ChoreID: id, OccurredAt: occAt}); err != nil {
		tx.Rollback()
		return fmt.Errorf("inserting new event: %w", err)
	}
	if err := txc.CompleteChore(ctx, cdb.CompleteChoreParams{ID: id, LastCompletion: occAt}); err != nil {
		tx.Rollback()
		return fmt.Errorf("updating last completion: %w", err)
	}
	return nil
}

func Snooze(ctx context.Context, db *sql.DB, id string, snoozeFor date.Duration) error {
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
		tx.Rollback()
		return fmt.Errorf("can't snooze a chore that is not due: %s", durToNext)
	}
	snoozedFor := durToNext + snoozeFor
	srvu.GetLogger(ctx).Printf("(%s - %s)+%s=%s", ex.NextCompletion(), date.Today(), snoozeFor, snoozedFor)
	if snoozedFor < 0 {
		snoozedFor = 0
	}
	txc := cdb.New(tx)
	if err := txc.SnoozeChore(ctx, cdb.SnoozeChoreParams{ID: id, SnoozedFor: int64(snoozedFor)}); err != nil {
		tx.Rollback()
		return fmt.Errorf("updating snooze duration: %w", err)
	}
	return nil
}
