package core

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/SimonSchneider/chore-tracker/internal/cdb"
	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/goslu/sqlu"
	"net/http"
	"strconv"
)

type Input struct {
	Name        string
	ChoreType   string
	ChoreListID string
	Interval    date.Duration
	Repeats     int64
	Link        string
	Date        date.Date
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
	i.ChoreType = r.FormValue("choreType")
	if err := parse(&i.Interval, date.ParseDuration, r.FormValue("interval"), 0); err != nil {
		return fmt.Errorf("invalid interval: %w", err)
	}
	if err := parse(&i.Repeats, parseInt, r.FormValue("repeats"), -1); err != nil {
		return fmt.Errorf("invalid repeats: %w", err)
	}
	if err := parse(&i.Date, date.ParseDate, r.FormValue("date"), 0); err != nil {
		return fmt.Errorf("invalid date: %w", err)
	}
	i.Link = r.FormValue("link")
	return nil
}

func (i *Input) Validate(prev *Chore) error {
	if i.Name == "" {
		return fmt.Errorf("illegal empty name for new chore")
	}
	if prev == nil && i.ChoreListID == "" {
		return fmt.Errorf("illegal empty choreListID for new chore")
	}
	if prev != nil {
		i.ChoreType = prev.ChoreType
	}
	switch i.ChoreType {
	case ChoreTypeInterval:
		if !i.Date.IsZero() {
			return fmt.Errorf("interval chore can't have a date")
		}
		if i.Interval.Zero() {
			return fmt.Errorf("interval chore can't have a zero interval")
		}
		if !(i.Repeats == -1 || i.Repeats > 0) {
			return fmt.Errorf("interval chore must have repeats > 0 or -1")
		}
		if prev != nil {
			i.Date = prev.LastCompletion
		}
	case ChoreTypeOneshot:
		if !i.Date.IsZero() {
			return fmt.Errorf("oneshot chore can't have a date")
		}
		if !i.Interval.Zero() {
			return fmt.Errorf("oneshot chore can't have an interval")
		}
		if i.Repeats != 1 {
			return fmt.Errorf("oneshot chore can't have repeats")
		}
		if prev != nil {
			i.Date = prev.LastCompletion
		}
	case ChoreTypeDate:
		if i.Date.IsZero() {
			return fmt.Errorf("date chore must have a date")
		}
		if !i.Interval.Zero() {
			return fmt.Errorf("date chore can't have an interval: %s", i.Interval)
		}
		if i.Repeats != 1 {
			return fmt.Errorf("date chore can't have repeats: %d", i.Repeats)
		}
	default:
		return fmt.Errorf("illegal choreType: %s", i.ChoreType)
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
	if err := input.Validate(nil); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}
	row, err := cdb.New(db).CreateChore(ctx, cdb.CreateChoreParams{
		ID:             NewId(),
		Name:           input.Name,
		ChoreType:      input.ChoreType,
		CreatedAt:      int64(today),
		ChoreListID:    input.ChoreListID,
		Interval:       int64(input.Interval),
		LastCompletion: int64(input.Date),
		RepeatsLeft:    input.Repeats,
		CreatedBy:      userID,
		Link:           sqlu.NullString(input.Link),
	})
	if err != nil {
		return nil, fmt.Errorf("creating chore: %w", err)
	}
	chore := ChoreFromDb(row)
	return &chore, nil
}

func Update(ctx context.Context, db *sql.DB, prev *Chore, input Input) (*Chore, error) {
	if prev == nil || prev.ID == "" {
		return nil, fmt.Errorf("illegal empty id for updating chore")
	}
	if err := input.Validate(prev); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}
	dbChore, err := cdb.New(db).UpdateChore(ctx, cdb.UpdateChoreParams{
		ID:             prev.ID,
		Name:           input.Name,
		Interval:       int64(input.Interval),
		RepeatsLeft:    input.Repeats,
		LastCompletion: int64(input.Date),
		Link:           sqlu.NullString(input.Link),
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
