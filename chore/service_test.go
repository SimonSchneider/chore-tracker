package chore

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/SimonSchneider/go-testing/date"
	"github.com/SimonSchneider/go-testing/srvu"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestSnoozing(t *testing.T) {
	ctx := srvu.ContextWithLogger(context.Background(), srvu.NewLoggerFunc(t.Logf))
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()
	if err := Setup(ctx, db); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 5 day interval
	ex, err := Create(ctx, db, Input{Name: "simon", Interval: 5})
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	// complete 8 days ago
	if err := Complete(ctx, db, ex.ID, date.Today().Add(-8*date.Day)); err != nil {
		t.Fatalf("failed: %v", err)
	}
	// 2 days overdue
	fmt.Printf("completed: %s\n", ex.DurationToNext())
	// snoozed for 1 day (from now)
	var snoozDays date.Duration = 2
	if err := Snooze(ctx, db, ex.ID, snoozDays); err != nil {
		t.Fatalf("failed: %v", err)
	}
	// -2 + 5 = 3 from now
	// snooze for 1 day
	// -2 + 3 + 5 = 6 from now
	ch, err := Get(ctx, db, ex.ID)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	// should be due in 1 day
	expDate := date.Today().Add(snoozDays * date.Day)
	if ch.NextCompletion() != expDate {
		t.Fatalf("unexpected next completion: %s ex %s (lc %s - sn %s)", ch.NextCompletion(), expDate, ch.LastCompletion, ch.SnoozedFor)
	}
}
