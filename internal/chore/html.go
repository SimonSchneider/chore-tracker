package chore

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/SimonSchneider/chore-tracker/internal/cdb"
	"github.com/SimonSchneider/chore-tracker/pkg/auth"
	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/goslu/srvu"
	"github.com/SimonSchneider/goslu/templ"
	"net/http"
	"time"
)

func HandlerIndex(db *sql.DB, tmpls templ.TemplateProvider) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return RenderFrontPage(ctx, w, tmpls, db, date.Today())
	})
}

func HandlerAdd(db *sql.DB, tmpls templ.TemplateProvider) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var inp Input
		if err := srvu.Decode(r, &inp, false); err != nil {
			return err
		}
		if _, err := Create(ctx, db, date.Today(), inp); err != nil {
			return srvu.Err(http.StatusInternalServerError, fmt.Errorf("creating the chore: %w", err))
		}
		return RenderListView(ctx, w, tmpls, db, date.Today())
	})
}

func HtmlUpdate(db *sql.DB, tmpls templ.TemplateProvider) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
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
		return RenderListView(ctx, w, tmpls, db, date.Today())
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

func HanderGet(db *sql.DB, tmpls templ.TemplateProvider) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		if id == "" {
			return srvu.Err(http.StatusBadRequest, fmt.Errorf("missing id"))
		}
		ch, err := Get(ctx, db, id)
		if err != nil {
			return srvu.Err(http.StatusBadRequest, fmt.Errorf("getting chore from request: %w", err))
		}
		return tmpls.ExecuteTemplate(w, "chore-element.gohtml", ch)
	})
}

func HandlerEdit(db *sql.DB, tmpls templ.TemplateProvider) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		if id == "" {
			return srvu.Err(http.StatusBadRequest, fmt.Errorf("missing id"))
		}
		ch, err := Get(ctx, db, id)
		if err != nil {
			return srvu.Err(http.StatusBadRequest, fmt.Errorf("getting chore from request: %w", err))
		}
		return tmpls.ExecuteTemplate(w, "chore-modal.gohtml", ch)
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

func HtmlComplete(db *sql.DB, tmpls templ.TemplateProvider) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
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
		return RenderListView(ctx, w, tmpls, db, today)
	})
}

func HtmlSnooze(db *sql.DB, tmpls templ.TemplateProvider) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		today := date.Today()
		if id == "" {
			return srvu.Err(http.StatusBadRequest, fmt.Errorf("missing id"))
		}
		if err := Snooze(ctx, db, today, id, 1*date.Day); err != nil {
			return srvu.Err(http.StatusInternalServerError, fmt.Errorf("snoozing the chore: %w", err))
		}
		return RenderListView(ctx, w, tmpls, db, today)
	})
}

func HtmlExpedite(db *sql.DB, tmpls templ.TemplateProvider) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		today := date.Today()
		if id == "" {
			return srvu.Err(http.StatusBadRequest, fmt.Errorf("missing id"))
		}
		if err := Expedite(ctx, db, today, id); err != nil {
			return srvu.Err(http.StatusInternalServerError, fmt.Errorf("snoozing the chore: %w", err))
		}
		return RenderListView(ctx, w, tmpls, db, today)
	})
}

func HandlerNew(tmpls templ.TemplateProvider) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return tmpls.ExecuteTemplate(w, "chore-modal.gohtml", Chore{})
	})
}

func ChoreListPage(db *sql.DB, tmpls templ.TemplateProvider) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//choreListId := r.PathValue("choreListID")
		//if choreListId == "" {
		//	return srvu.Err(http.StatusBadRequest, fmt.Errorf("missing choreListID"))
		//}
		//cl, err := cdb.New(db).GetChoreList(ctx, choreListId)
		//if err != nil {
		//	return srvu.Err(http.StatusInternalServerError, fmt.Errorf("getting chore list: %w", err))
		//}
		//return tmpls.ExecuteTemplate(w, "chore-list.page.gohtml", cl)
		return nil
	})
}

func ChoreListNewPage(tmpls templ.TemplateProvider) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return tmpls.ExecuteTemplate(w, "chore-list-new.page.gohtml", nil)
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
		http.Redirect(w, r, fmt.Sprintf("/chore-list/%s", cl.ID), http.StatusSeeOther)
		return nil
	})
}

func HtmlMux(db *sql.DB, tmplProvider templ.TemplateProvider) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("GET /chore-list/new", ChoreListNewPage(tmplProvider))
	mux.Handle("POST /chore-list/new", ChoreListNewHandler(db))
	mux.Handle("GET /chore-list/{choreListID}/{$}", ChoreListPage(db, tmplProvider))
	mux.Handle("GET /chore-list/{$}", ChoreListsPage(db, tmplProvider))
	mux.Handle("GET /{$}", HandlerIndex(db, tmplProvider))
	mux.Handle("GET /new", HandlerNew(tmplProvider))
	mux.Handle("GET /{id}/edit", HandlerEdit(db, tmplProvider))
	mux.Handle("POST /{$}", HandlerAdd(db, tmplProvider))
	mux.Handle("POST /{id}/complete", HtmlComplete(db, tmplProvider))
	mux.Handle("POST /{id}/snooze", HtmlSnooze(db, tmplProvider))
	mux.Handle("POST /{id}/expedite", HtmlExpedite(db, tmplProvider))
	mux.Handle("GET /{id}", HanderGet(db, tmplProvider))
	mux.Handle("DELETE /{id}", HandlerDelete(db))
	mux.Handle("PUT /{id}", HtmlUpdate(db, tmplProvider))
	return mux
}
