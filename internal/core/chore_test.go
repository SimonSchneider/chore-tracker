package core_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/SimonSchneider/chore-tracker/internal/core"
	"github.com/SimonSchneider/goslu/date"
)

func TestCRUDChore(t *testing.T) {
	ctx, client, cancel := Setup()
	defer cancel()
	tok := Must(client.NewToken(ctx))
	cl := Must(NewChoreList(ctx, client, tok, map[string]string{"name": "test"}))
	t.Run("Create Interval chore", func(t *testing.T) {
		chore, err := NewChore(ctx, client, tok, map[string]string{
			"name":        "interval",
			"interval":    "1w",
			"choreType":   core.ChoreTypeInterval,
			"choreListID": cl.List.ID,
		})
		if err != nil {
			t.Fatalf("failed to create chore: %s", err)
		}
		if chore.Name != "interval" {
			t.Fatalf("chore name is not 'test'")
		}
		if chore.Interval != date.Week {
			t.Fatalf("chore interval is not 1w")
		}
		if chore.ChoreListID != cl.List.ID {
			t.Fatalf("chore list id is not the same")
		}
		t.Run("Update", func(t *testing.T) {
			if _, err := NewChoreReq(ctx, client).Auth(tok).Form("POST", fmt.Sprintf("/chores/%s", chore.ID), map[string]string{
				"name":     "int2",
				"interval": "2w",
			}).DoAndFollow(http.StatusSeeOther); err != nil {
				t.Fatalf("failed to change chore: %s", err)
			}
			updatedChore := Must(GetChore(ctx, client, tok, cl.List.ID, chore.ID))
			if updatedChore.Name != "int2" {
				t.Fatalf("chore name is not 'int2'")
			}
			if updatedChore.Interval != 2*date.Week {
				t.Fatalf("chore interval is not 2w")
			}
		})
	})
	t.Run("Create Oneshot chore", func(t *testing.T) {
		chore, err := NewChore(ctx, client, tok, map[string]string{
			"name":        "oneshot",
			"repeats":     "1",
			"choreType":   core.ChoreTypeOneshot,
			"choreListID": cl.List.ID,
		})
		if err != nil {
			t.Fatalf("failed to create chore: %s", err)
		}
		if chore.Name != "oneshot" {
			t.Fatalf("chore name is not 'test'")
		}
		if chore.RepeatsLeft != 1 {
			t.Fatalf("chore repeats is not 1")
		}
		if chore.ChoreListID != cl.List.ID {
			t.Fatalf("chore list id is not the same")
		}
		t.Run("Update", func(t *testing.T) {
			if _, err := NewChoreReq(ctx, client).Auth(tok).Form("POST", fmt.Sprintf("/chores/%s", chore.ID), map[string]string{
				"name":    "once2",
				"repeats": "1",
			}).DoAndFollow(http.StatusSeeOther); err != nil {
				t.Fatalf("failed to change chore: %s", err)
			}
			updatedChore := Must(GetChore(ctx, client, tok, cl.List.ID, chore.ID))
			if updatedChore.Name != "once2" {
				t.Fatalf("chore name is not 'once2'")
			}
		})
	})
	t.Run("Create Date chore", func(t *testing.T) {
		chore, err := NewChore(ctx, client, tok, map[string]string{
			"name":        "date",
			"date":        date.Today().String(),
			"repeats":     "1",
			"choreType":   core.ChoreTypeDate,
			"choreListID": cl.List.ID,
		})
		if err != nil {
			t.Fatalf("failed to create chore: %s", err)
		}
		if chore.Name != "date" {
			t.Fatalf("chore name is not 'test'")
		}
		if chore.RepeatsLeft != 1 {
			t.Fatalf("chore repeats is not 1")
		}
		if chore.ChoreListID != cl.List.ID {
			t.Fatalf("chore list id is not the same")
		}
		t.Run("Update", func(t *testing.T) {
			if _, err := NewChoreReq(ctx, client).Auth(tok).Form("POST", fmt.Sprintf("/chores/%s", chore.ID), map[string]string{
				"name":    "date2",
				"repeats": "1",
				"date":    date.Today().Add(1).String(),
			}).DoAndFollow(http.StatusSeeOther); err != nil {
				t.Fatalf("failed to change chore: %s", err)
			}
			updatedChore := Must(GetChore(ctx, client, tok, cl.List.ID, chore.ID))
			if updatedChore.Name != "date2" {
				t.Fatalf("chore name is not 'test'")
			}
		})
	})
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
			"choreType":   core.ChoreTypeInterval,
			"choreListID": cl.List.ID,
			"interval":    Must(date.ParseDuration(fmt.Sprintf("1w%dd", len(createdChrs)-i))).String(),
		}))
		Must(NewChoreReq(ctx, client).Auth(tok).Form("POST", fmt.Sprintf("/chores/%s/complete", ch.ID), nil).DoAndFollow(http.StatusSeeOther))
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
