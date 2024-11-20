package chore

import (
	"context"
	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/goslu/srvu"
	"github.com/SimonSchneider/goslu/templ"
	"net/http"
	"sort"
	"time"
)

type ListView struct {
	Chores []Chore
}

func NewListView(chores []Chore) *ListView {
	sort.Slice(chores, func(i, j int) bool {
		return chores[i].NextCompletion().Before(chores[j].NextCompletion())
	})
	return &ListView{Chores: chores}
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
			if v.Chores[j].DurationToNext() <= sections[i].LatestCompletion {
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

func RenderListView(ctx context.Context, w http.ResponseWriter, tmpls templ.TemplateProvider, db Queryer) error {
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

func RenderFrontPage(ctx context.Context, w http.ResponseWriter, tmpls templ.TemplateProvider, db Queryer) error {
	chores, err := List(ctx, db)
	if err != nil {
		return srvu.Err(http.StatusInternalServerError, err)
	}
	return tmpls.ExecuteTemplate(w, "front-page.gohtml", FrontPage{
		Weekday: time.Now().Weekday(),
		Chores:  NewListView(chores),
	})
}
