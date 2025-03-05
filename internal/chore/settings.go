package chore

import (
	"context"
	"database/sql"
	"github.com/SimonSchneider/chore-tracker/internal/cdb"
	"github.com/SimonSchneider/chore-tracker/pkg/auth"
	"github.com/SimonSchneider/goslu/srvu"
	"github.com/SimonSchneider/goslu/templ"
	"net/http"
	"time"
)

type SettingsView struct {
	UserID         string
	Usernames      []string
	ChoreLists     []cdb.ChoreList
	CreatedInvites []cdb.GetInvitationsByCreatorRow
}

func SettingsPage(tmpls templ.TemplateProvider, db *sql.DB) http.Handler {
	q := cdb.New(db)
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		userId := auth.MustGetUserID(ctx)
		usernames, err := q.GetPasswordAuthsByUser(ctx, userId)
		if err != nil {
			return srvu.Err(http.StatusInternalServerError, err)
		}
		choreLists, err := q.GetChoreListsByUser(ctx, userId)
		if err != nil {
			return srvu.Err(http.StatusInternalServerError, err)
		}
		invites, err := q.GetInvitationsByCreator(ctx, cdb.GetInvitationsByCreatorParams{CreatedBy: userId, ExpiresAt: time.Now().UnixMilli()})
		return tmpls.ExecuteTemplate(w, "settings.page.gohtml", &SettingsView{
			UserID:         userId,
			Usernames:      usernames,
			ChoreLists:     choreLists,
			CreatedInvites: invites,
		})
	})
}
