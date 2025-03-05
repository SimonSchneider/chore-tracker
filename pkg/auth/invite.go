package auth

import (
	"context"
	"fmt"
	"github.com/SimonSchneider/goslu/srvu"
	"net/http"
	"time"
)

type InviteStore interface {
	CreateInvitePage(ctx context.Context, userID string, w http.ResponseWriter, r *http.Request) error
	CreateInvite(ctx context.Context, userID string, now time.Time, r *http.Request) (string, error)
	InvitePage(ctx context.Context, userID string, inviteID string, now time.Time, w http.ResponseWriter, r *http.Request) error
	InviteAccept(ctx context.Context, userID string, inviteID string, now time.Time, w http.ResponseWriter, r *http.Request) error
}

func inviteCreatePage(s InviteStore, cfg Config) http.Handler {
	return cfg.Middleware(false)(srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		userID := MustGetUserID(ctx)
		if err := s.CreateInvitePage(ctx, userID, w, r); err != nil {
			return srvu.Err(http.StatusInternalServerError, err)
		}
		return nil
	}))
}

func inviteCreateHandler(s InviteStore, cfg Config) http.Handler {
	return cfg.Middleware(false)(srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		userID := MustGetUserID(ctx)
		now := time.Now()
		invID, err := s.CreateInvite(ctx, userID, now, r)
		if err != nil {
			return srvu.Err(http.StatusInternalServerError, err)
		}
		http.Redirect(w, r, fmt.Sprintf("/invite/%s", invID), http.StatusFound)
		return nil
	}))
}

func invitePage(s InviteStore, cfg Config) http.Handler {
	return cfg.Middleware(true)(srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		inviteID := r.PathValue("inviteID")
		userID, _ := GetUserID(ctx)
		return s.InvitePage(ctx, userID, inviteID, time.Now(), w, r)
	}))
}

func inviteAcceptHandler(s InviteStore, cfg Config) http.Handler {
	return cfg.Middleware(true)(srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		userID, _ := GetUserID(ctx)
		inviteID := r.PathValue("inviteID")
		now := time.Now()
		return s.InviteAccept(ctx, userID, inviteID, now, w, r)
	}))
}

func InviteHandler(s InviteStore, authConfig Config) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /create", inviteCreatePage(s, authConfig))
	mux.Handle("POST /create", inviteCreateHandler(s, authConfig))
	mux.Handle("GET /{inviteID}", invitePage(s, authConfig))
	mux.Handle("POST /{inviteID}", inviteAcceptHandler(s, authConfig))
	return mux
}
