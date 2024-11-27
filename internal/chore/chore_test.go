package chore_test

import (
	"context"
	"database/sql"
	"fmt"
	choretracker "github.com/SimonSchneider/chore-tracker"
	"github.com/SimonSchneider/chore-tracker/internal/chore"
	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/goslu/srvu"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"log"
	"os"
	"testing"
)

func Panic(err error) {
	if err != nil {
		panic(err)
	}
}

func Must[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}
	return t
}

func Setup() (context.Context, *sql.DB, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	db, err := chore.GetMigratedDB(ctx, choretracker.StaticEmbeddedFS, "static/migrations", ":memory:")
	if err != nil {
		panic(fmt.Sprintf("failed to create test db: %s", err))
	}
	ctx = srvu.ContextWithLogger(ctx, srvu.LogToOutput(log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)))
	return ctx, db, cancel
}

func TestCreateChore(t *testing.T) {
	ctx, db, cancel := Setup()
	defer cancel()
	chr, err := chore.Create(ctx, db, chore.Input{Name: "test", Interval: Must(date.ParseDuration("1w"))})
	if err != nil {
		t.Fatalf("failed to create chore: %v", err)
	}
	if chr.ID == "" {
		t.Fatalf("chore id is empty")
	}
	if chr.Name != "test" {
		t.Fatalf("chore name is not 'test'")
	}
	if chr.Interval != Must(date.ParseDuration("1w")) {
		t.Fatalf("chore interval is not 1w")
	}
}

func TestGetChore(t *testing.T) {
	ctx, db, cancel := Setup()
	defer cancel()
	createdChr := Must(chore.Create(ctx, db, chore.Input{Name: "test", Interval: Must(date.ParseDuration("1w"))}))
	getChore, err := chore.Get(ctx, db, createdChr.ID)
	if err != nil {
		t.Fatalf("failed to get chore: %v", err)
	}
	if createdChr.ID != getChore.ID {
		t.Fatalf("chore id is not the same")
	}
	if createdChr.Name != getChore.Name {
		t.Fatalf("chore name is not the same")
	}
	if createdChr.Interval != getChore.Interval {
		t.Fatalf("chore interval is not the same")
	}
}

func TestListViewChores(t *testing.T) {
	ctx, db, cancel := Setup()
	defer cancel()
	createdChrs := make([]*chore.Chore, 5)
	for i := 0; i < len(createdChrs); i++ {
		chr := Must(chore.Create(ctx, db, chore.Input{Name: fmt.Sprintf("test%d", i), Interval: Must(date.ParseDuration(fmt.Sprintf("1w%dd", len(createdChrs)-i)))}))
		Panic(chore.Complete(ctx, db, chr.ID, date.Today()))
		createdChrs[i] = chr
	}
	list, err := chore.List(ctx, db)
	if err != nil {
		t.Fatalf("failed to list chores: %v", err)
	}
	view := chore.NewListView(list)
	for i, chr := range view.Chores {
		ci := len(createdChrs) - i - 1
		if chr.ID != createdChrs[ci].ID {
			t.Fatalf("chore id is not the same")
		}
		if chr.Name != createdChrs[ci].Name {
			t.Fatalf("chore name is not the same")
		}
		if chr.Interval != createdChrs[ci].Interval {
			t.Fatalf("chore interval is not the same")
		}
	}
}

func TestCompleteChore(t *testing.T) {
	ctx, db, cancel := Setup()
	defer cancel()
	createdChr := Must(chore.Create(ctx, db, chore.Input{Name: "test", Interval: Must(date.ParseDuration("1w"))}))
	err := chore.Complete(ctx, db, createdChr.ID, date.Today())
	if err != nil {
		t.Fatalf("failed to complete chore: %v", err)
	}
	chr := Must(chore.Get(ctx, db, createdChr.ID))
	if chr.LastCompletion.IsZero() {
		t.Fatalf("last completion is zero")
	}
	if chr.LastCompletion != date.Today() {
		t.Fatalf("last completion is not today")
	}
}

func TestSnoozeUncompleted(t *testing.T) {
	ctx, db, cancel := Setup()
	defer cancel()
	createdChr := Must(chore.Create(ctx, db, chore.Input{Name: "test", Interval: Must(date.ParseDuration("1w"))}))
	err := chore.Snooze(ctx, db, createdChr.ID, 1*date.Day)
	if err != nil {
		t.Fatalf("failed to snooze chore: %v", err)
	}
	chr := Must(chore.Get(ctx, db, createdChr.ID))
	if chr.SnoozedFor != 1*date.Day {
		t.Fatalf("snoozed for is not 1 day")
	}
	if chr.NextCompletion() != date.Today().Add(1*date.Day) {
		t.Fatalf("next completion is not 1 day from today")
	}
	if chr.DurationToNext() != 1*date.Day {
		t.Fatalf("duration to next is not 1 day")
	}
}

func TestSnoozeCompleted(t *testing.T) {
	ctx, db, cancel := Setup()
	defer cancel()
	createdChr := Must(chore.Create(ctx, db, chore.Input{Name: "test", Interval: Must(date.ParseDuration("1w"))}))
	Panic(chore.Complete(ctx, db, createdChr.ID, date.Today().Add(-1*date.Week-1*date.Day)))
	complChr := Must(chore.Get(ctx, db, createdChr.ID))
	if complChr.NextCompletion() != date.Today().Add(-1*date.Day) {
		t.Fatalf("last completion is not 1 week and 1 day ago")
	}
	err := chore.Snooze(ctx, db, createdChr.ID, 1*date.Day)
	if err != nil {
		t.Fatalf("failed to snooze chore: %v", err)
	}
	chr := Must(chore.Get(ctx, db, createdChr.ID))
	if chr.SnoozedFor != 2*date.Day {
		t.Fatalf("snoozed for is not 1 day: %s", chr.SnoozedFor)
	}
	if chr.NextCompletion() != date.Today().Add(1*date.Day) {
		t.Fatalf("next completion is not 1 day from today")
	}
	if chr.DurationToNext() != 1*date.Day {
		t.Fatalf("duration to next is not 1 day")
	}
}

func TestSnoozingANotYetDueChore(t *testing.T) {
	ctx, db, cancel := Setup()
	defer cancel()
	createdChr := Must(chore.Create(ctx, db, chore.Input{Name: "test", Interval: Must(date.ParseDuration("1w"))}))
	Panic(chore.Complete(ctx, db, createdChr.ID, date.Today()))
	err := chore.Snooze(ctx, db, createdChr.ID, 1*date.Day)
	if err == nil {
		t.Fatalf("snoozing a not yet due chore should fail")
	}
}

func TestExpediteChore(t *testing.T) {
	ctx, db, cancel := Setup()
	defer cancel()
	createdChr := Must(chore.Create(ctx, db, chore.Input{Name: "test", Interval: Must(date.ParseDuration("1w"))}))
	Panic(chore.Complete(ctx, db, createdChr.ID, date.Today()))
	Panic(chore.Expedite(ctx, db, createdChr.ID))
	chr := Must(chore.Get(ctx, db, createdChr.ID))
	if chr.NextCompletion() != date.Today() {
		t.Fatalf("next completion is not today: %s", chr.NextCompletion())
	}
}

func TestSnoozeAndExpedite(t *testing.T) {
	ctx, db, cancel := Setup()
	defer cancel()
	createdChr := Must(chore.Create(ctx, db, chore.Input{Name: "test", Interval: Must(date.ParseDuration("1w"))}))
	Panic(chore.Snooze(ctx, db, createdChr.ID, 1*date.Day))
	Panic(chore.Expedite(ctx, db, createdChr.ID))
	chr := Must(chore.Get(ctx, db, createdChr.ID))
	if chr.NextCompletion() != date.Today() {
		t.Fatalf("next completion is not today: %s", chr.NextCompletion())
	}
}
