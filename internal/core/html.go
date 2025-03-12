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
		userID := auth.MustGetUserID(ctx)
		var inp Input
		if err := srvu.Decode(r, &inp, false); err != nil {
			return err
		}
		chore, err := Create(ctx, db, date.Today(), userID, inp)
		if err != nil {
			return srvu.Err(http.StatusInternalServerError, fmt.Errorf("creating the chore: %w", err))
		}
		if r.Header.Get("HX-Request") == "true" {
			return ChoreListRender(ctx, db, view, w, r, date.Today(), userID, chore.ChoreListID)
		} else {
			http.Redirect(w, r, fmt.Sprintf("/chores/%s", chore.ID), http.StatusCreated)
			return nil
		}
	})
}

func ChoreEditPage(db *sql.DB, view *View) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		userID := auth.MustGetUserID(ctx)
		if id == "" {
			return srvu.Err(http.StatusBadRequest, fmt.Errorf("missing id"))
		}
		ch, err := Get(ctx, db, userID, id)
		if err != nil {
			return srvu.Err(http.StatusBadRequest, fmt.Errorf("getting chore from request: %w", err))
		}
		return view.ChoreModal(w, r, ch)
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
		userID := auth.MustGetUserID(ctx)
		today := date.Today()
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
		return ChoreListRender(ctx, db, view, w, r, today, userID, chore.ChoreListID)
	})
}

func ChoreSnoozeHandler(db *sql.DB, view *View) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		userID := auth.MustGetUserID(ctx)
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
		return ChoreListRender(ctx, db, view, w, r, today, userID, chore.ChoreListID)
	})
}

func ChoreExpediteHandler(db *sql.DB, view *View) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		userID := auth.MustGetUserID(ctx)
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
		return ChoreListRender(ctx, db, view, w, r, today, userID, chore.ChoreListID)
	})
}

func ChoreUpdateHandler(db *sql.DB, view *View) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		userID := auth.MustGetUserID(ctx)
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
		return ChoreListRender(ctx, db, view, w, r, date.Today(), userID, chore.ChoreListID)
	})
}

func ChorePage(db *sql.DB, view *View) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		userID := auth.MustGetUserID(ctx)
		if id == "" {
			return srvu.Err(http.StatusBadRequest, fmt.Errorf("missing id"))
		}
		ch, err := Get(ctx, db, userID, id)
		if err != nil {
			return srvu.Err(http.StatusBadRequest, fmt.Errorf("getting chore from request: %w", err))
		}
		return view.ChoreElement(w, r, ch)
	})
}

func ChoreDeleteHandler(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		userID := auth.MustGetUserID(ctx)
		if id == "" {
			return srvu.Err(http.StatusBadRequest, fmt.Errorf("missing id"))
		}
		_, err := Get(ctx, db, userID, id)
		if err != nil {
			return srvu.Err(http.StatusInternalServerError, fmt.Errorf("getting chore from request: %w", err))
		}
		if err := Delete(ctx, db, id); err != nil {
			return srvu.Err(http.StatusInternalServerError, fmt.Errorf("deleting the chore: %w", err))
		}
		w.WriteHeader(http.StatusOK)
		return nil
	})
}

func ChoreMux(db *sql.DB, view *View) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /chores/{id}/edit", ChoreEditPage(db, view))
	mux.Handle("POST /chores/{id}/complete", ChoreCompleteHandler(db, view))
	mux.Handle("POST /chores/{id}/snooze", ChoreSnoozeHandler(db, view))
	mux.Handle("POST /chores/{id}/expedite", ChoreExpediteHandler(db, view))
	mux.Handle("POST /chores/{$}", ChoreAddHandler(db, view))
	mux.Handle("GET /chores/{id}", ChorePage(db, view))
	mux.Handle("PUT /chores/{id}", ChoreUpdateHandler(db, view))
	mux.Handle("DELETE /chores/{id}", ChoreDeleteHandler(db))
	return mux
}
