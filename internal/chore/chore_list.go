package chore

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/SimonSchneider/chore-tracker/internal/cdb"
	"github.com/SimonSchneider/chore-tracker/pkg/auth"
	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/goslu/srvu"
	"io"
	"net/http"
	"time"
)

func ChoreListNewPage(tmpls *Templates) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return tmpls.ChoreListNewPage(w)
	})
}

func ChoreListNewHandler(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		userId := auth.MustGetUserID(ctx)
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
			UserID:      userId,
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

func ChoreListsPage(db *sql.DB, tmpls *Templates) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		userID := auth.MustGetUserID(ctx)
		choreLists, err := cdb.New(db).GetChoreListsByUser(ctx, userID)
		if err != nil {
			return srvu.Err(http.StatusInternalServerError, err)
		}
		return tmpls.ChoreListsPage(w, ChoreListsView{
			ChoreLists: choreLists,
		})
	})
}

func ChoreListRender(ctx context.Context, db *sql.DB, tmpls *Templates, w io.Writer, r *http.Request, today date.Date, userID, choreListID string) error {
	choreList, err := cdb.New(db).GetChoreListByUser(ctx, cdb.GetChoreListByUserParams{ID: choreListID, UserID: userID})
	if err != nil {
		return srvu.Err(http.StatusInternalServerError, err)
	}
	chores, err := cdb.New(db).GetChoresByList(ctx, choreListID)
	if err != nil {
		return srvu.Err(http.StatusInternalServerError, err)
	}
	view := ChoreListView{
		List:    choreList,
		Weekday: time.Now().Weekday(),
		Chores:  NewListView(today, ChoresFromDb(chores)),
	}
	if r.Header.Get("HX-Request") == "true" {
		return tmpls.ChoreList(w, view)
	}
	return tmpls.ChoreListPage(w, view)
}

func ChoreListPage(db *sql.DB, tmpls *Templates) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		choreListID := r.PathValue("choreListID")
		userID := auth.MustGetUserID(ctx)
		return ChoreListRender(ctx, db, tmpls, w, r, date.Today(), userID, choreListID)
	})
}

func ChoreListMux(db *sql.DB, tmplProvider *Templates) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("GET /chore-lists/new", ChoreListNewPage(tmplProvider))
	mux.Handle("POST /chore-lists/new", ChoreListNewHandler(db))
	mux.Handle("GET /chore-lists/{choreListID}/chores/new", ChoreNewPage(tmplProvider))
	mux.Handle("GET /chore-lists/{choreListID}/{$}", ChoreListPage(db, tmplProvider))
	mux.Handle("GET /chore-lists/{$}", ChoreListsPage(db, tmplProvider))
	return mux
}
