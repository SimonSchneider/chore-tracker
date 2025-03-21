package core

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/SimonSchneider/chore-tracker/pkg/auth"
	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/goslu/srvu"
	"net/http"
)

func ChoreAddHandler(db *sql.DB, view *View) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		userID := auth.MustGetSession(ctx).UserID
		var inp Input
		if err := srvu.Decode(r, &inp, false); err != nil {
			return err
		}
		chore, err := Create(ctx, db, date.Today(), userID, inp)
		if err != nil {
			return srvu.Err(http.StatusInternalServerError, fmt.Errorf("creating the chore: %w", err))
		}
		redirectToNext(w, r, fmt.Sprintf("/chore-lists/%s", chore.ChoreListID))
		return nil
	})
}

func ChoreEditPage(db *sql.DB, view *View) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		userID := auth.MustGetSession(ctx).UserID
		if id == "" {
			return srvu.Err(http.StatusBadRequest, fmt.Errorf("missing id"))
		}
		ch, err := Get(ctx, db, userID, id)
		if err != nil {
			return srvu.Err(http.StatusBadRequest, fmt.Errorf("getting chore from request: %w", err))
		}
		choreType := r.FormValue("chore-type")
		if choreType == "" {
			choreType = ch.ChoreType()
		}
		return view.ChoreEditPage(w, r, ChoreEditView{Chore: *ch, ChoreType: choreType})
	})
}

type CompletionInput struct {
	CompletedAt date.Date `json:"completed_at"`
}

func (c *CompletionInput) FromForm(r *http.Request) (err error) {
	val := r.FormValue("completed_at")
	if val == "" {
		return nil
	}
	c.CompletedAt, err = date.ParseDate(val)
	if err != nil {
		return fmt.Errorf("illegal completed_at '%s': %w", r.FormValue("completed_at"), err)
	}
	return nil
}

func ChoreCompleteHandler(db *sql.DB, view *View) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		userID := auth.MustGetSession(ctx).UserID
		if id == "" {
			return srvu.Err(http.StatusBadRequest, fmt.Errorf("missing id"))
		}
		var inp CompletionInput
		if err := srvu.Decode(r, &inp, false); err != nil {
			return srvu.Err(http.StatusBadRequest, fmt.Errorf("decoding input: %w", err))
		}
		chore, err := Get(ctx, db, userID, id)
		if err != nil {
			return srvu.Err(http.StatusInternalServerError, fmt.Errorf("getting chore from request: %w", err))
		}
		if err := Complete(ctx, db, userID, id, inp.CompletedAt); err != nil {
			return srvu.Err(http.StatusInternalServerError, fmt.Errorf("completing the chore: %w", err))
		}
		redirectToNext(w, r, fmt.Sprintf("/chore-lists/%s", chore.ChoreListID))
		return nil
	})
}

func ChoreSnoozeHandler(db *sql.DB, view *View) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		userID := auth.MustGetSession(ctx).UserID
		today := date.Today()
		if id == "" {
			return srvu.Err(http.StatusBadRequest, fmt.Errorf("missing id"))
		}
		chore, err := Get(ctx, db, userID, id)
		if err != nil {
			return srvu.Err(http.StatusInternalServerError, fmt.Errorf("getting chore from request: %w", err))
		}
		if err := Snooze(ctx, db, today, userID, id, 1*date.Day); err != nil {
			return srvu.Err(http.StatusInternalServerError, fmt.Errorf("snoozing the chore: %w", err))
		}
		redirectToNext(w, r, fmt.Sprintf("/chore-lists/%s", chore.ChoreListID))
		return nil
	})
}

func ChoreExpediteHandler(db *sql.DB, view *View) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		userID := auth.MustGetSession(ctx).UserID
		today := date.Today()
		if id == "" {
			return srvu.Err(http.StatusBadRequest, fmt.Errorf("missing id"))
		}
		chore, err := Get(ctx, db, userID, id)
		if err != nil {
			return srvu.Err(http.StatusInternalServerError, fmt.Errorf("getting chore from request: %w", err))
		}
		if err := Expedite(ctx, db, today, userID, id); err != nil {
			return srvu.Err(http.StatusInternalServerError, fmt.Errorf("snoozing the chore: %w", err))
		}
		redirectToNext(w, r, fmt.Sprintf("/chore-lists/%s", chore.ChoreListID))
		return nil
	})
}

func ChoreUpdateHandler(db *sql.DB, view *View) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		userID := auth.MustGetSession(ctx).UserID
		if id == "" {
			return srvu.Err(http.StatusBadRequest, fmt.Errorf("missing id"))
		}
		var inp Input
		if err := srvu.Decode(r, &inp, false); err != nil {
			return srvu.Err(http.StatusBadRequest, fmt.Errorf("decoding input: %w", err))
		}
		chore, err := Get(ctx, db, userID, id)
		if err != nil {
			return srvu.Err(http.StatusInternalServerError, fmt.Errorf("getting chore from request: %w", err))
		}
		if _, err := Update(ctx, db, id, inp); err != nil {
			return srvu.Err(http.StatusInternalServerError, fmt.Errorf("updating the chore: %w", err))
		}
		redirectToNext(w, r, fmt.Sprintf("/chore-lists/%s", chore.ChoreListID))
		return nil
	})
}

func ChoreDeleteHandler(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		userID := auth.MustGetSession(ctx).UserID
		if id == "" {
			return srvu.Err(http.StatusBadRequest, fmt.Errorf("missing id"))
		}
		ch, err := Get(ctx, db, userID, id)
		if err != nil {
			return srvu.Err(http.StatusInternalServerError, fmt.Errorf("getting chore from request: %w", err))
		}
		if err := Delete(ctx, db, id); err != nil {
			return srvu.Err(http.StatusInternalServerError, fmt.Errorf("deleting the chore: %w", err))
		}
		redirectToNext(w, r, fmt.Sprintf("/chore-lists/%s", ch.ChoreListID))
		return nil
	})
}

func ChoreMux(db *sql.DB, view *View) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /chores/{id}/edit", ChoreEditPage(db, view))
	mux.Handle("POST /chores/{id}/complete", ChoreCompleteHandler(db, view))
	mux.Handle("POST /chores/{id}/snooze", ChoreSnoozeHandler(db, view))
	mux.Handle("POST /chores/{id}/expedite", ChoreExpediteHandler(db, view))
	mux.Handle("POST /chores/{id}/delete", ChoreDeleteHandler(db))
	mux.Handle("POST /chores/{$}", ChoreAddHandler(db, view))
	mux.Handle("POST /chores/{id}", ChoreUpdateHandler(db, view))
	mux.Handle("DELETE /chores/{id}", ChoreDeleteHandler(db))
	return mux
}
