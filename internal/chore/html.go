package chore

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/SimonSchneider/chore-tracker/pkg/auth"
	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/goslu/srvu"
	"net/http"
)

func ChoreAddHandler(db *sql.DB, tmpls *Templates) http.Handler {
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
		return ChoreListRender(ctx, db, tmpls, w, r, date.Today(), userID, chore.ChoreListID)
	})
}

func HtmlUpdate(db *sql.DB, tmpls *Templates) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		userID := auth.MustGetUserID(ctx)
		if id == "" {
			return srvu.Err(http.StatusBadRequest, fmt.Errorf("missing id"))
		}
		fmt.Printf("id: %s\n", id)
		var inp Input
		if err := srvu.Decode(r, &inp, false); err != nil {
			return srvu.Err(http.StatusBadRequest, fmt.Errorf("decoding input: %w", err))
		}
		if err := Update(ctx, db, id, inp); err != nil {
			return srvu.Err(http.StatusInternalServerError, fmt.Errorf("updating the chore: %w", err))
		}
		// TODO: fix
		return ChoreListRender(ctx, db, tmpls, w, r, date.Today(), userID, "")
	})
}

func HandlerDelete(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		if id == "" {
			return srvu.Err(http.StatusBadRequest, fmt.Errorf("missing id"))
		}
		if err := Delete(ctx, db, id); err != nil {
			return srvu.Err(http.StatusInternalServerError, fmt.Errorf("deleting the chore: %w", err))
		}
		w.WriteHeader(http.StatusOK)
		return nil
	})
}

func HanderGet(db *sql.DB, tmpls *Templates) http.Handler {
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
		return tmpls.ChoreElement(w, ch)
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

func ChoreCompleteHandler(db *sql.DB, tmpls *Templates) http.Handler {
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
		if err := Complete(ctx, db, id, inp.CompletedAt); err != nil {
			return srvu.Err(http.StatusInternalServerError, fmt.Errorf("completing the chore: %w", err))
		}
		// TODO: fix
		return ChoreListRender(ctx, db, tmpls, w, r, today, userID, "")
	})
}

func ChoreSnoozeHandler(db *sql.DB, tmpls *Templates) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		userID := auth.MustGetUserID(ctx)
		today := date.Today()
		if id == "" {
			return srvu.Err(http.StatusBadRequest, fmt.Errorf("missing id"))
		}
		if err := Snooze(ctx, db, today, userID, id, 1*date.Day); err != nil {
			return srvu.Err(http.StatusInternalServerError, fmt.Errorf("snoozing the chore: %w", err))
		}
		// TODO: fix
		return ChoreListRender(ctx, db, tmpls, w, r, today, userID, "")
	})
}

func ChoreExpediteHandler(db *sql.DB, tmpls *Templates) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		userID := auth.MustGetUserID(ctx)
		today := date.Today()
		if id == "" {
			return srvu.Err(http.StatusBadRequest, fmt.Errorf("missing id"))
		}
		if err := Expedite(ctx, db, today, userID, id); err != nil {
			return srvu.Err(http.StatusInternalServerError, fmt.Errorf("snoozing the chore: %w", err))
		}
		// TODO: fix
		return ChoreListRender(ctx, db, tmpls, w, r, today, userID, "")
	})
}

func ChoreNewPage(tmpls *Templates) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		choreListID := r.PathValue("choreListID")
		return tmpls.ChoreModal(w, &Chore{
			ChoreListID: choreListID,
		})
	})
}

func ChoreEditPage(db *sql.DB, tmpls *Templates) http.Handler {
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
		return tmpls.ChoreModal(w, ch)
	})
}

func ChoreMux(db *sql.DB, tmpls *Templates) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("POST /chores/new", ChoreAddHandler(db, tmpls))
	mux.Handle("GET /chores/{id}/edit", ChoreEditPage(db, tmpls))
	mux.Handle("POST /chores/{id}/complete", ChoreCompleteHandler(db, tmpls))
	mux.Handle("POST /chores/{id}/snooze", ChoreSnoozeHandler(db, tmpls))
	mux.Handle("POST /chores/{id}/expedite", ChoreExpediteHandler(db, tmpls))
	mux.Handle("GET /chores/{id}", HanderGet(db, tmpls))
	mux.Handle("DELETE /chores/{id}", HandlerDelete(db))
	mux.Handle("PUT /chores/{id}", HtmlUpdate(db, tmpls))
	return mux
}
