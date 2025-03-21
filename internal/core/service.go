package core

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/SimonSchneider/chore-tracker/internal/cdb"
	"github.com/SimonSchneider/goslu/date"
	"net/http"
	"strconv"
)

type Input struct {
	Name        string
	ChoreListID string
	Interval    date.Duration
	Repeats     int64
}

func parse[T any](into *T, parser func(string) (T, error), val string, ifEmpty T) error {
	if val == "" {
		*into = ifEmpty
		return nil
	}
	parsed, err := parser(val)
	if err != nil {
		return fmt.Errorf("illegal value '%s': %w", val, err)
	}
	*into = parsed
	return nil
}

func parseInt(val string) (int64, error) {
	return strconv.ParseInt(val, 10, 64)
}

func (i *Input) FromForm(r *http.Request) error {
	i.Name = r.FormValue("name")
	i.ChoreListID = r.FormValue("choreListID")
	if err := parse(&i.Interval, date.ParseDuration, r.FormValue("interval"), 0); err != nil {
		return fmt.Errorf("invalid interval: %w", err)
	}
	if err := parse(&i.Repeats, parseInt, r.FormValue("repeats"), -1); err != nil {
		return fmt.Errorf("invalid repeats: %w", err)
	}
	return nil
}

func ChoresFromDb(dbChores []cdb.Chore) []Chore {
	chores := make([]Chore, len(dbChores))
	for i, dbChore := range dbChores {
		chores[i] = ChoreFromDb(dbChore)
	}
	return chores
}

func Get(ctx context.Context, db cdb.DBTX, userID, id string) (*Chore, error) {
	return get(ctx, cdb.New(db), userID, id)
}

func get(ctx context.Context, db *cdb.Queries, userID, id string) (*Chore, error) {
	row, err := db.GetChore(ctx, cdb.GetChoreParams{ID: id, UserID: userID})
	if err != nil {
		return nil, fmt.Errorf("querying chore %s for user %s: %w", id, userID, err)
	}
	chore := ChoreFromDb(row)
	return &chore, nil
}

func Create(ctx context.Context, db *sql.DB, today date.Date, userID string, input Input) (*Chore, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("illegal empty name for new chore")
	}
	if input.Interval.Zero() && input.Repeats < 1 {
		return nil, fmt.Errorf("chore interval can't be zero if there are no repeats")
	}
	row, err := cdb.New(db).CreateChore(ctx, cdb.CreateChoreParams{
		ID:          NewId(),
		Name:        input.Name,
		CreatedAt:   int64(today),
		ChoreListID: input.ChoreListID,
		Interval:    int64(input.Interval),
		RepeatsLeft: input.Repeats,
		CreatedBy:   userID,
	})
	if err != nil {
		return nil, fmt.Errorf("creating chore: %w", err)
	}
	chore := ChoreFromDb(row)
	return &chore, nil
}

func Update(ctx context.Context, db *sql.DB, id string, input Input) (*Chore, error) {
	if id == "" {
		return nil, fmt.Errorf("illegal empty id for updating chore")
	}
	if input.Name == "" || input.Interval.Zero() {
		return nil, fmt.Errorf("illegal empty name or interval for updating chore")
	}
	dbChore, err := cdb.New(db).UpdateChore(ctx, cdb.UpdateChoreParams{
		ID:          id,
		Name:        input.Name,
		Interval:    int64(input.Interval),
		RepeatsLeft: input.Repeats,
	})
	if err != nil {
		return nil, fmt.Errorf("updating chore: %w", err)
	}
	chore := ChoreFromDb(dbChore)
	return &chore, nil
}

func Delete(ctx context.Context, db *sql.DB, id string) error {
	if err := cdb.New(db).DeleteChore(ctx, id); err != nil {
		return fmt.Errorf("deleting chore: %w", err)
	}
	return nil
}

func Complete(ctx context.Context, db *sql.DB, userID, id string, occurredAt date.Date) error {
	// TODO: idempotency (with etag versioning?)
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
	if err := txc.CreateChoreEvent(ctx, cdb.CreateChoreEventParams{ID: NewId(), ChoreID: id, OccurredAt: occAt, CreatedBy: userID, EventType: "complete"}); err != nil {
		tx.Rollback()
		return fmt.Errorf("inserting new event: %w", err)
	}
	if err := txc.CompleteChore(ctx, cdb.CompleteChoreParams{ID: id, LastCompletion: occAt}); err != nil {
		tx.Rollback()
		return fmt.Errorf("updating last completion: %w", err)
	}
	return nil
}

func Expedite(ctx context.Context, db *sql.DB, today date.Date, userID, id string) error {
	return changeSnooze(ctx, db, today, userID, id, 0, func(durToNext date.Duration) bool {
		return durToNext <= 0
	})
}

func Snooze(ctx context.Context, db *sql.DB, today date.Date, userID, id string, snoozeFor date.Duration) error {
	return changeSnooze(ctx, db, today, userID, id, snoozeFor, func(durToNext date.Duration) bool {
		return durToNext > 0
	})
}

func changeSnooze(ctx context.Context, db *sql.DB, today date.Date, userID, id string, snoozeFor date.Duration, validateDurToNext func(date.Duration) bool) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning tx: %w", err)
	}
	txc := cdb.New(tx)
	defer tx.Commit()
	ex, err := get(ctx, txc, userID, id)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("getting chore: %w", err)
	}
	durToNext := ex.DurationToNextFrom(today)
	if validateDurToNext(durToNext) {
		tx.Rollback()
		return fmt.Errorf("can't snooze a chore that is not due: %s", durToNext)
	}
	snoozedFor := snoozeFor + ex.SnoozedFor - durToNext
	if err := txc.SnoozeChore(ctx, cdb.SnoozeChoreParams{ID: id, SnoozedFor: int64(snoozedFor)}); err != nil {
		tx.Rollback()
		return fmt.Errorf("updating snooze duration: %w", err)
	}
	return nil
}
