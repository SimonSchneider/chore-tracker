package core

import (
	"github.com/SimonSchneider/goslu/date"
	"sort"
)

type ListView struct {
	Chores []Chore
	Today  date.Date
}

func NewListView(today date.Date, chores []Chore) *ListView {
	sort.Slice(chores, func(i, j int) bool {
		return chores[i].NextCompletion().Before(chores[j].NextCompletion())
	})
	return &ListView{Chores: chores, Today: today}
}

func (v *ListView) Sections() []Section {
	sections := []Section{
		{Title: "Overdue", LatestCompletion: -1 * date.Day},
		{Title: "Today", LatestCompletion: date.Zero},
		{Title: "Tomorrow", LatestCompletion: date.Day},
		{Title: "This week", LatestCompletion: date.Week},
		{Title: "This month", LatestCompletion: 1 * date.Month},
		{Title: "Later", LatestCompletion: date.Max},
	}
	j := 0
	for i := range sections {
		for ; j < len(v.Chores); j++ {
			if v.Chores[j].DurationToNextFrom(v.Today) <= sections[i].LatestCompletion {
				sections[i].Chores = append(sections[i].Chores, v.Chores[j])
			} else {
				break
			}
		}
	}
	return sections
}

type Section struct {
	Title            string
	LatestCompletion date.Duration
	Chores           []Chore
}

func (s *Section) HasChores() bool {
	return len(s.Chores) > 0
}

func (s *Section) IsOpen() bool {
	return s.HasChores() && s.LatestCompletion <= date.Week
}
