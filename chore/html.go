package chore

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/SimonSchneider/go-testing/srvu"
	"net/http"
)

func HtmlList(db *sql.DB, tmpls TemplateProvider) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		chores, err := List(ctx, db)
		if err != nil {
			return srvu.Err(http.StatusInternalServerError, err)
		}
		tmpl := tmpls.Lookup("list-chores.gohtml")
		return tmpl.Execute(w, chores)
	})
}

func HtmlAdd(db *sql.DB, tmpls TemplateProvider) http.Handler {
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
		if _, err := Create(ctx, tx, inp); err != nil {
			tx.Rollback()
			return srvu.Err(http.StatusInternalServerError, fmt.Errorf("creating the chore: %w", err))
		}
		tx.Commit()
		chores, err := List(ctx, db)
		if err != nil {
			return srvu.Err(http.StatusInternalServerError, fmt.Errorf("listing chores: %w", err))
		}
		tmpl := tmpls.Lookup("list-chores.gohtml")
		return tmpl.Execute(w, chores)
	})
}

func NewHtmlMux(db *sql.DB, tmplProvider TemplateProvider) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("GET /", HtmlList(db, tmplProvider))
	mux.Handle("POST /{$}", HtmlAdd(db, tmplProvider))
	return mux
}
