package chore

import (
	"github.com/SimonSchneider/go-testing/date"
)

type Chore struct {
	ID             string        `json:"id,omitempty"`
	Name           string        `json:"name,omitempty"`
	Interval       date.Duration `json:"interval,omitempty"`
	LastCompletion date.Date     `json:"last_completion,omitempty"`
	SnoozedFor     date.Duration `json:"znoozed_for,omitempty"`
	History        []Event       `json:"history,omitempty"`
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

type Event struct {
	ID         string    `json:"id,omitempty"`
	OccurredAt date.Date `json:"occurred_at,omitempty"`
}
