package core

import (
	"context"
	"database/sql"
	"github.com/SimonSchneider/chore-tracker/internal/cdb"
	"github.com/SimonSchneider/chore-tracker/pkg/auth"
	"github.com/SimonSchneider/goslu/srvu"
	"net/http"
	"time"
)

func SettingsPage(view *View, db *sql.DB) http.Handler {
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
		return view.SettingsPage(w, r, SettingsView{
			UserID:         userId,
			Usernames:      usernames,
			ChoreLists:     choreLists,
			CreatedInvites: invites,
		})
	})
}
