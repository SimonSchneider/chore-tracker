package core

import (
	"github.com/SimonSchneider/chore-tracker/internal/cdb"
	"github.com/SimonSchneider/goslu/date"
)

type Chore struct {
	ID             string
	Name           string
	CreatedAt      date.Date
	Interval       date.Duration
	LastCompletion date.Date
	SnoozedFor     date.Duration
	ChoreListID    string
	RepeatsLeft    int64 // -1 means infinite
}

func (c *Chore) Repeats() bool {
	return c.RepeatsLeft > 0
}

func (c *Chore) ChoreType() string {
	if c.Interval == 0 {
		return "one-time"
	}
	return "repeating"
}

func (c *Chore) NextCompletion() date.Date {
	if c.LastCompletion.IsZero() {
		return c.CreatedAt.Add(c.SnoozedFor)
	}
	return c.LastCompletion.Add(c.Interval + c.SnoozedFor)
}

func (c *Chore) DurationToNextFrom(today date.Date) date.Duration {
	return c.NextCompletion().Sub(today)
}

func (c *Chore) DurationToNext() date.Duration {
	return c.NextCompletion().Sub(date.Today())
}

func ChoreFromDb(row cdb.Chore) Chore {
	return Chore{
		ID:             row.ID,
		Name:           row.Name,
		CreatedAt:      date.Date(row.CreatedAt),
		Interval:       date.Duration(row.Interval),
		LastCompletion: date.Date(row.LastCompletion),
		SnoozedFor:     date.Duration(row.SnoozedFor),
		ChoreListID:    row.ChoreListID,
		RepeatsLeft:    row.RepeatsLeft,
	}
}
