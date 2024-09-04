package chore

import (
	"context"
	"github.com/SimonSchneider/go-testing/duration"
	"github.com/SimonSchneider/go-testing/srvu"
	"net/http"
	"sort"
	"time"
)

type ListView struct {
	chores []Chore
}

func NewListView(chores []Chore) *ListView {
	sort.Slice(chores, func(i, j int) bool {
		return chores[i].NextCompletion().Before(chores[j].NextCompletion())
	})
	return &ListView{chores: chores}
}

func (v *ListView) Sections() []Section {
	sections := []Section{
		{Title: "Overdue", latestCompletion: -1 * duration.Day},
		{Title: "Today", latestCompletion: duration.Zero},
		{Title: "Tomorrow", latestCompletion: duration.Day},
		{Title: "This week", latestCompletion: duration.Week},
		{Title: "This month", latestCompletion: 1 * duration.Month},
		{Title: "Later", latestCompletion: duration.Max},
	}
	j := 0
	for i := range sections {
		for ; j < len(v.chores); j++ {
			if v.chores[j].DurationToNext() <= sections[i].latestCompletion {
				sections[i].Chores = append(sections[i].Chores, v.chores[j])
			} else {
				break
			}
		}
	}
	return sections
}

type Section struct {
	Title            string
	latestCompletion duration.Duration
	Chores           []Chore
}

func (s *Section) HasChores() bool {
	return len(s.Chores) > 0
}

func RenderListView(ctx context.Context, w http.ResponseWriter, tmpls srvu.TemplateProvider, db Queryer) error {
	chores, err := List(ctx, db)
	if err != nil {
		return srvu.Err(http.StatusInternalServerError, err)
	}
	return tmpls.ExecuteTemplate(w, "chore-list.gohtml", NewListView(chores))
}

type FrontPage struct {
	Weekday time.Weekday
	Chores  *ListView
}

func RenderFrontPage(ctx context.Context, w http.ResponseWriter, tmpls srvu.TemplateProvider, db Queryer) error {
	chores, err := List(ctx, db)
	if err != nil {
		return srvu.Err(http.StatusInternalServerError, err)
	}
	return tmpls.ExecuteTemplate(w, "front-page.gohtml", FrontPage{
		Weekday: time.Now().Weekday(),
		Chores:  NewListView(chores),
	})
}
