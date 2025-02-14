package chore

import (
	"github.com/SimonSchneider/chore-tracker/internal/cdb"
	"github.com/SimonSchneider/chore-tracker/internal/dateglob"
	"github.com/SimonSchneider/goslu/date"
)

type Chore struct {
	ID             string        `json:"id,omitempty"`
	Name           string        `json:"name,omitempty"`
	CreatedAt      date.Date     `json:"created_at,omitempty"`
	Interval       date.Duration `json:"interval,omitempty"`
	DateGlob       string        `json:"date_glob,omitempty"`
	LastCompletion date.Date     `json:"last_completion,omitempty"`
	SnoozedFor     date.Duration `json:"znoozed_for,omitempty"`
}

func (c *Chore) NextCompletion() date.Date {
	if c.DateGlob != "" {
		dg, err := dateglob.Parse(c.DateGlob)
		if err != nil {
			return date.Today()
		}
		prevOccurrence := c.LastCompletion
		if prevOccurrence.IsZero() {
			prevOccurrence = c.CreatedAt
		}
		prevOccurrence = prevOccurrence.Add(c.SnoozedFor)
		next, err := dg.NextMatch(prevOccurrence)
		if err != nil {
			return date.Today()
		}
		return next
	}
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
		DateGlob:       row.DateGlob.String,
		LastCompletion: date.Date(row.LastCompletion),
		SnoozedFor:     date.Duration(row.SnoozedFor),
	}
}
