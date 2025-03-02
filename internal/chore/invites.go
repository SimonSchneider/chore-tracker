package chore

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/SimonSchneider/chore-tracker/internal/auth"
	"github.com/SimonSchneider/chore-tracker/internal/cdb"
	"github.com/SimonSchneider/goslu/sqlu"
	"github.com/SimonSchneider/goslu/srvu"
	"github.com/SimonSchneider/goslu/templ"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"time"
)

type InviteCreateView struct {
	ChoreLists []cdb.ChoreList
}

func InviteCreatePage(tmpls templ.TemplateProvider, db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		userID := auth.MustGetUserID(ctx)
		choreLists, err := cdb.New(db).ChoreListsForUser(ctx, userID)
		if err != nil {
			return srvu.Err(http.StatusInternalServerError, err)
		}
		return tmpls.ExecuteTemplate(w, "invite_create.gohtml", InviteCreateView{ChoreLists: choreLists})
	})
}

func InviteCreateHandler(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		userID := auth.MustGetUserID(ctx)
		choreListID := r.FormValue("choreListID")
		if choreListID != "" {
			choreList, err := cdb.New(db).GetChoreListByUser(ctx, cdb.GetChoreListByUserParams{UserID: userID, ID: choreListID})
			if err != nil || choreList.ID == "" {
				return srvu.Err(http.StatusUnauthorized, err)
			}
		}
		now := time.Now()
		inv, err := cdb.New(db).CreateInvite(ctx, cdb.CreateInviteParams{
			ID:          NewId(),
			CreatedAt:   now.UnixMilli(),
			ExpiresAt:   now.Add(24 * time.Hour).UnixMilli(),
			ChoreListID: sqlu.NullString(choreListID),
			CreatedBy:   userID,
		})
		if err != nil {
			return srvu.Err(http.StatusInternalServerError, fmt.Errorf("create invite: %w", err))
		}
		http.Redirect(w, r, fmt.Sprintf("/invite/%s", inv.ID), http.StatusFound)
		return nil
	})
}

type InviteView struct {
	InviteID string
	BaseURL  string
}

func InvitePage(tmpls templ.TemplateProvider, db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		inviteID := r.PathValue("inviteID")
		if inviteID == "" {
			return srvu.Err(http.StatusBadRequest, fmt.Errorf("missing inviteID"))
		}
		userID := auth.MustGetUserID(ctx)
		invite, err := cdb.New(db).GetInvite(ctx, cdb.GetInviteParams{ID: inviteID, ExpiresAt: time.Now().UnixMilli()})
		if err != nil {
			return srvu.Err(http.StatusInternalServerError, fmt.Errorf("get invite: %w", err))
		}
		if invite.CreatedBy != userID {
			return srvu.Err(http.StatusUnauthorized, fmt.Errorf("not authorized to view invite"))
		}
		baseURL := getURL(r)
		return tmpls.ExecuteTemplate(w, "invite.gohtml", InviteView{
			InviteID: invite.ID,
			BaseURL:  baseURL,
		})
	})
}

type InviteAcceptView struct {
	InviteID      string
	ChoreListName string
	InviterName   string
	ExistingUser  bool
}

func InviteAcceptPage(tmpls templ.TemplateProvider, db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		inviteID := r.PathValue("inviteID")
		if inviteID == "" {
			return srvu.Err(http.StatusBadRequest, fmt.Errorf("missing inviteID"))
		}
		invite, err := cdb.New(db).GetInvite(ctx, cdb.GetInviteParams{ID: inviteID, ExpiresAt: time.Now().UnixMilli()})
		if err != nil {
			return srvu.Err(http.StatusInternalServerError, fmt.Errorf("get invite: %w", err))
		}
		userID, _ := auth.GetUserID(ctx)
		return tmpls.ExecuteTemplate(w, "invite_accept.gohtml", InviteAcceptView{
			InviteID:      invite.ID,
			ChoreListName: invite.ChoreListName.String,
			InviterName:   invite.CreatedByName.String,
			ExistingUser:  userID != "",
		})
	})
}

func InviteAcceptHandler(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		inviteID := r.FormValue("inviteID")
		username := r.FormValue("name")
		password := r.FormValue("password")
		if inviteID == "" {
			return srvu.Err(http.StatusBadRequest, fmt.Errorf("missing inviteID"))
		}
		userID, _ := auth.GetUserID(ctx)
		if userID == "" && (username == "" || password == "") {
			return srvu.Err(http.StatusBadRequest, fmt.Errorf("missing name or password for non logged in user"))
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return srvu.Err(http.StatusInternalServerError, fmt.Errorf("begin tx: %w", err))
		}
		defer tx.Rollback()
		q := cdb.New(tx)
		now := time.Now()
		invite, err := q.DeleteInvite(ctx, cdb.DeleteInviteParams{ID: inviteID, ExpiresAt: now.UnixMilli()})
		if err != nil || invite.ID == "" {
			return srvu.Err(http.StatusNotFound, fmt.Errorf("invalid invite: %s", inviteID))
		}
		if userID == "" {
			hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if err != nil {
				return srvu.Err(http.StatusInternalServerError, fmt.Errorf("hashing password: %w", err))
			}
			user, err := q.CreateUser(ctx, cdb.CreateUserParams{
				ID:        NewId(),
				CreatedAt: now.UnixMilli(),
				UpdatedAt: now.UnixMilli(),
			})
			if err != nil {
				return srvu.Err(http.StatusInternalServerError, fmt.Errorf("create user: %w", err))
			}
			if err := q.CreatePasswordAuth(ctx, cdb.CreatePasswordAuthParams{
				UserID:   user.ID,
				Username: username,
				Hash:     string(hash),
			}); err != nil {
				return srvu.Err(http.StatusInternalServerError, fmt.Errorf("add user password: %w", err))
			}
			userID = user.ID
		}
		if invite.ChoreListID.Valid {
			if err := q.AddUserToChoreList(ctx, cdb.AddUserToChoreListParams{
				UserID:      userID,
				ChoreListID: invite.ChoreListID.String,
			}); err != nil {
				return srvu.Err(http.StatusInternalServerError, fmt.Errorf("add user to chore list: %w", err))
			}
		}
		if err := tx.Commit(); err != nil {
			return srvu.Err(http.StatusInternalServerError, fmt.Errorf("commit tx: %w", err))
		}
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return nil
	})
}

func InviteMux(db *sql.DB, tmpls templ.TemplateProvider, authConfig auth.Config) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /create", srvu.With(InviteCreatePage(tmpls, db), authConfig.Middleware(false)))
	mux.Handle("POST /create", srvu.With(InviteCreateHandler(db), authConfig.Middleware(false)))
	mux.Handle("GET /{inviteID}", srvu.With(InvitePage(tmpls, db), authConfig.Middleware(false)))
	mux.Handle("GET /{inviteID}/accept", srvu.With(InviteAcceptPage(tmpls, db), authConfig.Middleware(true)))
	mux.Handle("POST /{inviteID}/accept", srvu.With(InviteAcceptHandler(db), authConfig.Middleware(true)))
	return mux
}

func getURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	// Check for forwarded headers (useful when behind a reverse proxy)
	if forwardedProto := r.Header.Get("X-Forwarded-Proto"); forwardedProto != "" {
		scheme = forwardedProto
	}

	// Get the host (domain + optional port)
	host := r.Host
	if forwardedHost := r.Header.Get("X-Forwarded-Host"); forwardedHost != "" {
		host = forwardedHost
	}

	// Construct the base URL
	return scheme + "://" + host
}
