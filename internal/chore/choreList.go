package chore

import (
	"context"
	"database/sql"
	"github.com/SimonSchneider/chore-tracker/internal/cdb"
	"github.com/SimonSchneider/chore-tracker/pkg/auth"
	"github.com/SimonSchneider/goslu/srvu"
	"github.com/SimonSchneider/goslu/templ"
	"net/http"
)

type ChoreListView struct {
	ChoreLists []cdb.ChoreList
}

func ChoreListsPage(db *sql.DB, tmpls templ.TemplateProvider) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		userID := auth.MustGetUserID(ctx)
		choreLists, err := cdb.New(db).GetChoreListsByUser(ctx, userID)
		if err != nil {
			return srvu.Err(http.StatusInternalServerError, err)
		}
		return tmpls.ExecuteTemplate(w, "chore_lists.page.gohtml", ChoreListView{ChoreLists: choreLists})
	})
}
