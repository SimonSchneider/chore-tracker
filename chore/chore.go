package chore

import (
	"github.com/SimonSchneider/go-testing/duration"
	"time"
)

type Chore struct {
	ID             string            `json:"id,omitempty"`
	Name           string            `json:"name,omitempty"`
	Interval       duration.Duration `json:"interval,omitempty"`
	LastCompletion time.Time         `json:"last_completion,omitempty"`
	History        []Event           `json:"history,omitempty"`
}

func (c *Chore) NextCompletion() time.Time {
	if c.LastCompletion.IsZero() {
		return time.Now()
	}
	return c.LastCompletion.Add(c.Interval.ToCompat())
}

func (c *Chore) DurationToNext() duration.Duration {
	if c.LastCompletion.IsZero() {
		return duration.Zero
	}
	return c.Interval - duration.Sub(c.LastCompletion, time.Now())
}

type Event struct {
	ID         string    `json:"id,omitempty"`
	OccurredAt time.Time `json:"occurred_at,omitempty"`
}
