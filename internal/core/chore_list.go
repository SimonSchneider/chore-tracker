package core

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/SimonSchneider/chore-tracker/internal/cdb"
	"github.com/SimonSchneider/chore-tracker/pkg/auth"
	"github.com/SimonSchneider/chore-tracker/pkg/httpu"
	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/goslu/sqlu"
	"github.com/SimonSchneider/goslu/srvu"
	"net/http"
	"sort"
	"strings"
	"time"
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

func ChoreListNewPage(view *View) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return view.ChoreListNewPage(w, r)
	})
}

func ChoreListNewHandler(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		userID := auth.MustGetSession(ctx).UserID
		name := r.FormValue("name")
		if name == "" {
			return srvu.Err(http.StatusBadRequest, fmt.Errorf("missing name"))
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return srvu.Err(http.StatusInternalServerError, fmt.Errorf("beginning tx: %w", err))
		}
		defer tx.Rollback()
		q := cdb.New(tx)
		now := time.Now()
		cl, err := q.CreateChoreList(ctx, cdb.CreateChoreListParams{
			ID:        NewId(),
			Name:      name,
			CreatedAt: now.UnixMilli(),
			UpdatedAt: now.UnixMilli(),
		})
		if err != nil {
			return srvu.Err(http.StatusInternalServerError, fmt.Errorf("creating chore list: %w", err))
		}
		if err := q.AddUserToChoreList(ctx, cdb.AddUserToChoreListParams{
			UserID:      userID,
			ChoreListID: cl.ID,
		}); err != nil {
			return srvu.Err(http.StatusInternalServerError, fmt.Errorf("adding user to chore list: %w", err))
		}
		if err := tx.Commit(); err != nil {
			return srvu.Err(http.StatusInternalServerError, fmt.Errorf("committing tx: %w", err))
		}
		http.Redirect(w, r, fmt.Sprintf("/chore-lists/%s", cl.ID), http.StatusSeeOther)
		return nil
	})
}

func ChoreListUpdateHandler(db *sql.DB, view *View) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		userID := auth.MustGetSession(ctx).UserID
		name := r.FormValue("name")
		id := r.PathValue("choreListID")
		if id == "" {
			return srvu.Err(http.StatusBadRequest, fmt.Errorf("missing id"))
		}
		if name == "" {
			return srvu.Err(http.StatusBadRequest, fmt.Errorf("missing name"))
		}
		q := cdb.New(db)
		_, err := q.UpdateChoreList(ctx, cdb.UpdateChoreListParams{ID: id, UpdatedAt: time.Now().UnixMilli(), Name: name, UserID: userID})
		if err != nil {
			return srvu.Err(http.StatusInternalServerError, err)
		}
		httpu.RedirectToNext(w, r, fmt.Sprintf("/chore-lists/%s", id))
		return nil
	})
}

func ChoreListLeaveHandler(db *sql.DB, view *View) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		userID := auth.MustGetSession(ctx).UserID
		id := r.PathValue("choreListID")
		if id == "" {
			return srvu.Err(http.StatusBadRequest, fmt.Errorf("missing id"))
		}
		q := cdb.New(db)
		if err := q.RemoveUserFromChoreList(ctx, cdb.RemoveUserFromChoreListParams{UserID: userID, ChoreListID: id}); err != nil {
			return srvu.Err(http.StatusInternalServerError, err)
		}
		httpu.RedirectToNext(w, r, "/chore-lists")
		return nil
	})
}

func ChoreListsRender(ctx context.Context, db *sql.DB, view *View, w http.ResponseWriter, r *http.Request, userID string) error {
	choreLists, err := cdb.New(db).GetChoreListsByUser(ctx, userID)
	if err != nil {
		return srvu.Err(http.StatusInternalServerError, err)
	}
	return view.ChoreListsPage(w, r, ChoreListsView{
		ChoreLists: choreLists,
	})
}

func ChoreListsPage(db *sql.DB, view *View) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		userID := auth.MustGetSession(ctx).UserID
		return ChoreListsRender(ctx, db, view, w, r, userID)
	})
}

func ChoreListRender(ctx context.Context, db *sql.DB, view *View, w http.ResponseWriter, r *http.Request, today date.Date, userID, choreListID string) error {
	choreList, err := cdb.New(db).GetChoreListByUser(ctx, cdb.GetChoreListByUserParams{ID: choreListID, UserID: userID})
	if err != nil {
		return srvu.Err(http.StatusInternalServerError, err)
	}
	chores, err := cdb.New(db).GetChoresByList(ctx, choreListID)
	if err != nil {
		return srvu.Err(http.StatusInternalServerError, err)
	}
	return view.ChoreListPage(w, r, ChoreListView{
		List:    choreList,
		Weekday: time.Now().Weekday(),
		Chores:  NewListView(today, ChoresFromDb(chores)),
	})
}

func ChoreListPage(db *sql.DB, view *View) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		choreListID := r.PathValue("choreListID")
		userID := auth.MustGetSession(ctx).UserID
		return ChoreListRender(ctx, db, view, w, r, date.Today(), userID, choreListID)
	})
}

func ChoreListEditPage(db *sql.DB, view *View) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		userID := auth.MustGetSession(ctx).UserID
		id := r.PathValue("choreListID")
		q := cdb.New(db)
		choreList, err := q.GetChoreListByUser(ctx, cdb.GetChoreListByUserParams{UserID: userID, ID: id})
		if err != nil {
			return srvu.Err(http.StatusInternalServerError, err)
		}
		members, err := q.GetChoreListMembers(ctx, choreList.ID)
		if err != nil {
			return srvu.Err(http.StatusInternalServerError, err)
		}
		invites, err := q.GetInvitationsByChoreList(ctx, cdb.GetInvitationsByChoreListParams{ChoreListID: sqlu.NullString(choreList.ID), ExpiresAt: time.Now().UnixMilli()})
		return view.ChoreListEditPage(w, r, ChoreListEditView{
			List:    choreList,
			Members: members,
			Invites: invites,
		})
	})
}

func ChoreListNewChorePage(view *View) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		choreListID := r.PathValue("choreListID")
		return view.ChoreCreatePage(w, r, ChoreEditView{
			Chore:     Chore{ChoreListID: choreListID},
			ChoreType: Coalesce(r.FormValue("chore-type"), "interval"),
		})
	})
}

func ChoreListCreateInviteHandler(db *sql.DB, view *View, inviteStore *InviteStore) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		userID := auth.MustGetSession(ctx).UserID
		choreListID := r.PathValue("choreListID")
		if choreListID == "" {
			return srvu.Err(http.StatusBadRequest, fmt.Errorf("missing id"))
		}
		_, err := inviteStore.CreateInviteWithChoreList(ctx, userID, choreListID, time.Now(), r)
		if err != nil {
			return srvu.Err(http.StatusInternalServerError, err)
		}
		httpu.RedirectToReferer(w, r, fmt.Sprintf("/chore-lists/%s/edit", choreListID))
		return nil
	})
}

func ChoreListDeleteInviteHandler(db *sql.DB, view *View, inviteStore *InviteStore) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		userID := auth.MustGetSession(ctx).UserID
		choreListID := r.PathValue("choreListID")
		inviteID := r.PathValue("inviteID")
		if inviteID == "" {
			return srvu.Err(http.StatusBadRequest, fmt.Errorf("missing inviteID"))
		}
		if choreListID == "" {
			return srvu.Err(http.StatusBadRequest, fmt.Errorf("missing choreListID"))
		}
		_, err := cdb.New(db).GetChoreListByUser(ctx, cdb.GetChoreListByUserParams{ID: choreListID, UserID: userID})
		if err != nil {
			return srvu.Err(http.StatusForbidden, err)
		}
		if err := inviteStore.DeleteInviteInChoreList(ctx, inviteID, choreListID); err != nil {
			return srvu.Err(http.StatusInternalServerError, err)
		}
		httpu.RedirectToReferer(w, r, fmt.Sprintf("/chore-lists/%s/edit", choreListID))
		return nil
	})
}

func ChoreListChartPage(db *sql.DB, view *View) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		choreListID := r.PathValue("choreListID")
		userID := auth.MustGetSession(ctx).UserID
		cl, err := cdb.New(db).GetChoreListByUser(ctx, cdb.GetChoreListByUserParams{ID: choreListID, UserID: userID})
		if err != nil {
			return srvu.Err(http.StatusForbidden, err)
		}
		return view.ChoreListChartPage(w, r, ChoreListChartView{
			List: cl,
		})
	})
}

func ChoreListChartDataHandler(db *sql.DB, view *View) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		choreListID := r.PathValue("choreListID")
		chartData := r.PathValue("chartType")
		userID := auth.MustGetSession(ctx).UserID
		if choreListID == "" {
			return srvu.Err(http.StatusBadRequest, fmt.Errorf("missing choreListID"))
		}
		if chartData == "" {
			return srvu.Err(http.StatusBadRequest, fmt.Errorf("missing chartData"))
		}
		switch chartData {
		case "completion_calendar":
			data, err := cdb.New(db).GetChoreListCalendarCompletionData(ctx, cdb.GetChoreListCalendarCompletionDataParams{
				ChoreListID: choreListID,
				UserID:      userID,
			})
			if err != nil {
				return srvu.Err(http.StatusForbidden, err)
			}
			cld := &ChoreListDataView{
				Data: make([]ChoreListDataViewSeries, 0, len(data)),
			}
			for _, d := range data {
				cld.Data = append(cld.Data, ChoreListDataViewSeries{
					Date:  date.Date(d.OccurredAt),
					Value: d.Count,
				})
			}
			return view.ChoreListChartData(w, r, cld)
		default:
			return srvu.Err(http.StatusBadRequest, fmt.Errorf("unknown chart type: %s", chartData))
		}
	})
}

func ChoreListMux(db *sql.DB, view *View, inviteStore *InviteStore) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("GET /chore-lists/new", ChoreListNewPage(view))
	mux.Handle("POST /chore-lists/", ChoreListNewHandler(db))
	mux.Handle("POST /chore-lists/{choreListID}", ChoreListUpdateHandler(db, view))
	mux.Handle("POST /chore-lists/{choreListID}/leave", ChoreListLeaveHandler(db, view))
	mux.Handle("POST /chore-lists/{choreListID}/invites/", ChoreListCreateInviteHandler(db, view, inviteStore))
	mux.Handle("POST /chore-lists/{choreListID}/invites/{inviteID}/delete", ChoreListDeleteInviteHandler(db, view, inviteStore))
	mux.Handle("GET /chore-lists/{choreListID}/chores/new", ChoreListNewChorePage(view))
	mux.Handle("GET /chore-lists/{choreListID}/edit", ChoreListEditPage(db, view))
	mux.Handle("GET /chore-lists/{choreListID}/charts", ChoreListChartPage(db, view))
	mux.Handle("GET /chore-lists/{choreListID}/charts/{chartType}", ChoreListChartDataHandler(db, view))
	mux.Handle("GET /chore-lists/{choreListID}", ChoreListPage(db, view))
	mux.Handle("GET /chore-lists/{choreListID}/", ChoreListPage(db, view))
	mux.Handle("GET /chore-lists/{$}", ChoreListsPage(db, view))
	return mux
}

func APIChoreListIcsFile(db *sql.DB, view *View, apiKey string) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		choreListID := r.PathValue("choreListID")
		qAPIKey := r.FormValue("apiKey")
		if qAPIKey == "" {
			return srvu.Err(http.StatusBadRequest, fmt.Errorf("missing APIKey"))
		}
		if apiKey == "" {
			return srvu.Err(http.StatusBadRequest, fmt.Errorf("missing APIKey"))
		}
		// TODO: improve api key handling and make it dynamic instead of hardcoded
		if strings.Compare(apiKey, choreListID) == 0 {
			return srvu.Err(http.StatusUnauthorized, fmt.Errorf("invalid APIKey"))
		}
		cl, err := cdb.New(db).GetChoreListWithoutUser(ctx, choreListID)
		if err != nil {
			return srvu.Err(http.StatusInternalServerError, err)
		}
		chores, err := cdb.New(db).GetChoresByList(ctx, choreListID)
		if err != nil {
			return srvu.Err(http.StatusInternalServerError, err)
		}
		return view.ChoreListIcs(w, r, &ChoreListIcsView{
			ID:     cl.ID,
			Name:   cl.Name,
			Today:  date.Today(),
			Chores: ChoresFromDb(chores),
		})
	})
}

func ChoreListAPIMux(db *sql.DB, view *View, apiKey string) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("GET /api/chore-lists/{choreListID}/ics", APIChoreListIcsFile(db, view, apiKey))
	return mux
}
