package chore

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/SimonSchneider/go-testing/srvu"
	"net/http"
	"time"
)

func HandlerAdd(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var inp Input
		if err := srvu.Decode(r, &inp, false); err != nil {
			return err
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return srvu.Err(http.StatusInternalServerError, fmt.Errorf("begin tx: %w", err))
		}
		defer tx.Commit()
		ch, err := Create(ctx, tx, inp)
		if err != nil {
			tx.Rollback()
			return srvu.Err(http.StatusInternalServerError, fmt.Errorf("creating the chore: %w", err))
		}
		return srvu.Encode(w, ch)
	})
}

func HandlerList(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		chores, err := List(ctx, db)
		if err != nil {
			return srvu.Err(http.StatusInternalServerError, err)
		}
		return srvu.Encode(w, chores)
	})
}

func HandlerGet(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		if id == "" {
			return srvu.ErrStr(http.StatusBadRequest, "missing id in path")
		}
		chore, err := Get(ctx, db, id)
		if err != nil {
			return srvu.Err(http.StatusInternalServerError, err)
		}
		return srvu.Encode(w, chore)
	})
}

func HandlerComplete(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		if id == "" {
			return srvu.ErrStr(http.StatusBadRequest, "missing id in path")
		}
		var inp struct {
			OccurredAt time.Time `json:"occurred_at"`
		}
		if err := srvu.Decode(r, &inp, true); err != nil {
			return err
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return srvu.Err(http.StatusInternalServerError, fmt.Errorf("begin tx: %w", err))
		}
		defer tx.Commit()
		ch, err := Complete(ctx, tx, id, inp.OccurredAt)
		if err != nil {
			tx.Rollback()
			return srvu.Err(http.StatusInternalServerError, fmt.Errorf("completing the chore: %w", err))
		}
		return srvu.Encode(w, ch)
	})
}

func HandlerDeleteCompletion(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		if id == "" {
			return srvu.ErrStr(http.StatusBadRequest, "missing id in path")
		}
		completionID := r.PathValue("completionId")
		if completionID == "" {
			return srvu.ErrStr(http.StatusBadRequest, "missing completion id in path")
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return srvu.Err(http.StatusInternalServerError, fmt.Errorf("begin tx: %w", err))
		}
		defer tx.Commit()
		if err := DeleteCompletion(ctx, tx, id, completionID); err != nil {
			tx.Rollback()
			return srvu.Err(http.StatusInternalServerError, fmt.Errorf("deleting the completion: %w", err))
		}
		w.WriteHeader(http.StatusNoContent)
		return nil
	})
}

func NewMux(db *sql.DB) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("POST /{$}", HandlerAdd(db))
	mux.Handle("GET /{id}", HandlerList(db))
	mux.Handle("POST /{id}/completion/{$}", HandlerComplete(db))
	mux.Handle("DELETE /{id}/completion/{completionId}", HandlerDeleteCompletion(db))
	return mux
}
