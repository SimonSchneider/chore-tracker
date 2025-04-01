package core_test

import (
	"fmt"
	"github.com/SimonSchneider/chore-tracker/internal/core"
	"github.com/SimonSchneider/goslu/date"
	"net/http"
	"testing"
)

func TestCreateChore(t *testing.T) {
	ctx, client, cancel := Setup()
	defer cancel()
	tok := Must(client.NewToken(ctx))
	cl := Must(NewChoreList(ctx, client, tok, map[string]string{"name": "test"}))
	chore, err := NewChore(ctx, client, tok, map[string]string{
		"name":        "test",
		"interval":    "1w",
		"choreListID": cl.List.ID,
	})
	if err != nil {
		t.Fatalf("failed to create chore: %s", err)
	}
	if chore.Name != "test" {
		t.Fatalf("chore name is not 'test'")
	}
	if chore.Interval != date.Week {
		t.Fatalf("chore interval is not 1w")
	}
	if chore.ChoreListID != cl.List.ID {
		t.Fatalf("chore list id is not the same")
	}
}

func TestChangeIntervalChore(t *testing.T) {
	ctx, client, cancel := Setup()
	defer cancel()
	tok := Must(client.NewToken(ctx))
	cl := Must(NewChoreList(ctx, client, tok, map[string]string{"name": "test"}))
	ch := Must(NewChore(ctx, client, tok, map[string]string{"name": "test", "interval": "1w", "choreListID": cl.List.ID}))
	if _, err := NewChoreReq(ctx, client).Auth(tok).Form("POST", fmt.Sprintf("/chores/%s", ch.ID), tok.CSRF, map[string]string{
		"name":     "test2",
		"interval": "2w",
	}).DoAndFollow(http.StatusSeeOther); err != nil {
		t.Fatalf("failed to change chore: %s", err)
	}
	updatedChore := Must(GetChore(ctx, client, tok, cl.List.ID, ch.ID))
	if updatedChore.Name != "test2" {
		t.Fatalf("chore name is not 'test'")
	}
	if updatedChore.Interval != 2*date.Week {
		t.Fatalf("chore interval is not 2w")
	}
}

func TestChangeOneshotChore(t *testing.T) {
	ctx, client, cancel := Setup()
	defer cancel()
	tok := Must(client.NewToken(ctx))
	cl := Must(NewChoreList(ctx, client, tok, map[string]string{"name": "test"}))
	ch := Must(NewChore(ctx, client, tok, map[string]string{"name": "test", "repeats": "1", "choreListID": cl.List.ID}))
	if _, err := NewChoreReq(ctx, client).Auth(tok).Form("POST", fmt.Sprintf("/chores/%s", ch.ID), tok.CSRF, map[string]string{
		"name":    "test2",
		"repeats": "1",
	}).DoAndFollow(http.StatusSeeOther); err != nil {
		t.Fatalf("failed to change chore: %s", err)
	}
	updatedChore := Must(GetChore(ctx, client, tok, cl.List.ID, ch.ID))
	if updatedChore.Name != "test2" {
		t.Fatalf("chore name is not 'test'")
	}
}

func TestListViewChores(t *testing.T) {
	ctx, client, cancel := Setup()
	defer cancel()
	tok := Must(client.NewToken(ctx))
	cl := Must(NewChoreList(ctx, client, tok, map[string]string{"name": "test"}))
	createdChrs := make([]*core.Chore, 5)
	for i := 0; i < len(createdChrs); i++ {
		ch := Must(NewChore(ctx, client, tok, map[string]string{
			"name":        "test",
			"choreListID": cl.List.ID,
			"interval":    Must(date.ParseDuration(fmt.Sprintf("1w%dd", len(createdChrs)-i))).String(),
		}))
		Must(NewChoreReq(ctx, client).Auth(tok).Form("POST", fmt.Sprintf("/chores/%s/complete", ch.ID), tok.CSRF, nil).DoAndFollow(http.StatusSeeOther))
		createdChrs[i] = ch
	}
	Must(NewChoreReq(ctx, client).Auth(tok).Get(fmt.Sprintf("/chore-lists/%s", cl.List.ID)).DoAndExp(http.StatusOK))
	clView := GetTpl[core.ChoreListView](client.tmpl, "chore_list.page.gohtml")
	for i, chr := range clView.Chores.Chores {
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

/*

   func TestCompleteChore(t *testing.T) {
   	ctx, db, cancel := Setup()
   	defer cancel()
   	createdChr := Must(core.Create(ctx, db, date.Today(), core.Input{Name: "test", Interval: Must(date.ParseDuration("1w"))}))
   	err := core.Complete(ctx, db, createdChr.ID, date.Today())
   	if err != nil {
   		t.Fatalf("failed to complete chore: %v", err)
   	}
   	chr := Must(core.Get(ctx, db, createdChr.ID))
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
   	today := date.Today()
   	createdChr := Must(core.Create(ctx, db, today, core.Input{Name: "test", Interval: Must(date.ParseDuration("1w"))}))
   	err := core.Snooze(ctx, db, today, createdChr.ID, 1*date.Day)
   	if err != nil {
   		t.Fatalf("failed to snooze chore: %v", err)
   	}
   	chr := Must(core.Get(ctx, db, createdChr.ID))
   	if chr.SnoozedFor != 1*date.Day {
   		t.Fatalf("snoozed for is not 1 day")
   	}
   	if chr.NextCompletion() != date.Today().Add(1*date.Day) {
   		t.Fatalf("next completion is not 1 day from today")
   	}
   	if chr.DurationToNextFrom(today) != 1*date.Day {
   		t.Fatalf("duration to next is not 1 day")
   	}
   	tomorrow := today.Add(1 * date.Day)
   	if chr.NextCompletion() != tomorrow {
   		t.Fatalf("next completion is not 0 day from tomorrow(%s): %s", tomorrow, chr.NextCompletion())
   	}
   	if chr.DurationToNextFrom(tomorrow) != 0 {
   		t.Fatalf("duration to next is not 0: %s", chr.DurationToNextFrom(tomorrow))
   	}
   }

   func TestSnoozeCompleted(t *testing.T) {
   	ctx, db, cancel := Setup()
   	defer cancel()
   	today := date.Today()
   	createdChr := Must(core.Create(ctx, db, today, core.Input{Name: "test", Interval: Must(date.ParseDuration("1w"))}))
   	Panic(core.Complete(ctx, db, createdChr.ID, date.Today().Add(-1*date.Week-1*date.Day)))
   	complChr := Must(core.Get(ctx, db, createdChr.ID))
   	if complChr.NextCompletion() != date.Today().Add(-1*date.Day) {
   		t.Fatalf("last completion is not 1 week and 1 day ago")
   	}
   	err := core.Snooze(ctx, db, today, createdChr.ID, 1*date.Day)
   	if err != nil {
   		t.Fatalf("failed to snooze chore: %v", err)
   	}
   	chr := Must(core.Get(ctx, db, createdChr.ID))
   	if chr.SnoozedFor != 2*date.Day {
   		t.Fatalf("snoozed for is not 1 day: %s", chr.SnoozedFor)
   	}
   	if chr.NextCompletion() != date.Today().Add(1*date.Day) {
   		t.Fatalf("next completion is not 1 day from today")
   	}
   	if chr.DurationToNextFrom(today) != 1*date.Day {
   		t.Fatalf("duration to next is not 1 day")
   	}
   	tomorrow := today.Add(1 * date.Day)
   	if chr.NextCompletion() != tomorrow {
   		t.Fatalf("next completion is not 0 day from tomorrow(%s): %s", tomorrow, chr.NextCompletion())
   	}
   	if chr.DurationToNextFrom(tomorrow) != 0 {
   		t.Fatalf("duration to next is not 0: %s", chr.DurationToNextFrom(tomorrow))
   	}
   }

   func TestSnoozingANotYetDueChore(t *testing.T) {
   	ctx, db, cancel := Setup()
   	defer cancel()
   	today := date.Today()
   	createdChr := Must(core.Create(ctx, db, today, core.Input{Name: "test", Interval: Must(date.ParseDuration("1w"))}))
   	Panic(core.Complete(ctx, db, createdChr.ID, date.Today()))
   	err := core.Snooze(ctx, db, today, createdChr.ID, 1*date.Day)
   	if err == nil {
   		t.Fatalf("snoozing a not yet due chore should fail")
   	}
   }

   func TestExpediteChore(t *testing.T) {
   	ctx, db, cancel := Setup()
   	defer cancel()
   	today := date.Today()
   	createdChr := Must(core.Create(ctx, db, today, core.Input{Name: "test", Interval: Must(date.ParseDuration("1w"))}))
   	Panic(core.Complete(ctx, db, createdChr.ID, today))
   	Panic(core.Expedite(ctx, db, today, createdChr.ID))
   	chr := Must(core.Get(ctx, db, createdChr.ID))
   	if chr.NextCompletion() != today {
   		t.Fatalf("next completion is not today: %s", chr.NextCompletion())
   	}
   }

   func TestSnoozeAndExpedite(t *testing.T) {
   	ctx, db, cancel := Setup()
   	defer cancel()
   	today := date.Today()
   	createdChr := Must(core.Create(ctx, db, today, core.Input{Name: "test", Interval: Must(date.ParseDuration("1w"))}))
   	Panic(core.Snooze(ctx, db, today, createdChr.ID, 1*date.Day))
   	Panic(core.Expedite(ctx, db, today, createdChr.ID))
   	chr := Must(core.Get(ctx, db, createdChr.ID))
   	if chr.NextCompletion() != today {
   		t.Fatalf("next completion is not today: %s", chr.NextCompletion())
   	}
   }
*/
