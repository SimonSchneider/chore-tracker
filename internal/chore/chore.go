package chore

import (
	"github.com/SimonSchneider/go-testing/internal/cdb"
	"github.com/SimonSchneider/goslu/date"
)

type Chore struct {
	ID             string        `json:"id,omitempty"`
	Name           string        `json:"name,omitempty"`
	Interval       date.Duration `json:"interval,omitempty"`
	LastCompletion date.Date     `json:"last_completion,omitempty"`
	SnoozedFor     date.Duration `json:"znoozed_for,omitempty"`
}

func (c *Chore) NextCompletion() date.Date {
	if c.LastCompletion.IsZero() {
		return date.Today().Add(c.SnoozedFor)
	}
	return c.LastCompletion.Add(c.Interval + c.SnoozedFor)
}

func (c *Chore) DurationToNext() date.Duration {
	if c.LastCompletion.IsZero() {
		return date.Zero
	}
	return c.NextCompletion().Sub(date.Today())
}

func ChoreFromDb(row cdb.Chore) Chore {
	return Chore{
		ID:             row.ID,
		Name:           row.Name,
		Interval:       date.Duration(row.Interval),
		LastCompletion: date.Date(row.LastCompletion),
		SnoozedFor:     date.Duration(row.SnoozedFor),
	}
}
