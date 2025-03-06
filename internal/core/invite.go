package core

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/SimonSchneider/chore-tracker/internal/cdb"
	"github.com/SimonSchneider/goslu/sqlu"
	"github.com/SimonSchneider/goslu/srvu"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"time"
)

type InviteStore struct {
	db   *sql.DB
	view *View
}

func (s *InviteStore) CreateInvitePage(ctx context.Context, userID string, w http.ResponseWriter, r *http.Request) error {
	choreLists, err := cdb.New(s.db).GetChoreListsByUser(ctx, userID)
	if err != nil {
		return err
	}
	return s.view.InviteCreate(w, r, InviteCreateView{ChoreLists: choreLists})
}

func (s *InviteStore) CreateInvite(ctx context.Context, userID string, now time.Time, r *http.Request) (string, error) {
	choreListID := r.FormValue("choreListID")
	if choreListID != "" {
		choreList, err := cdb.New(s.db).GetChoreListByUser(ctx, cdb.GetChoreListByUserParams{UserID: userID, ID: choreListID})
		if err != nil || choreList.ID == "" {
			return "", srvu.Err(http.StatusUnauthorized, err)
		}
	}
	inv, err := cdb.New(s.db).CreateInvite(ctx, cdb.CreateInviteParams{
		ID:          NewId(),
		CreatedAt:   now.UnixMilli(),
		ExpiresAt:   now.Add(24 * time.Hour).UnixMilli(),
		ChoreListID: sqlu.NullString(choreListID),
		CreatedBy:   userID,
	})
	if err != nil {
		return "", err
	}
	return inv.ID, err
}

func (s *InviteStore) InvitePage(ctx context.Context, userID string, inviteID string, now time.Time, w http.ResponseWriter, r *http.Request) error {
	invite, err := cdb.New(s.db).GetInvite(ctx, cdb.GetInviteParams{ID: inviteID, ExpiresAt: now.UnixMilli()})
	if err != nil {
		return err
	}
	if invite.CreatedBy == userID {

		return s.view.InvitePage(w, r, InviteView{
			InviteID:      invite.ID,
			ChoreListName: invite.ChoreListName.String,
		})
	}
	return s.view.InviteAcceptPage(w, r, InviteAcceptView{
		InviteID:      invite.ID,
		ChoreListName: invite.ChoreListName.String,
		InviterName:   invite.CreatedByName.String,
		ExistingUser:  userID != "",
	})
}

func (s *InviteStore) InviteAccept(ctx context.Context, userID string, inviteID string, now time.Time, w http.ResponseWriter, r *http.Request) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return srvu.Err(http.StatusInternalServerError, err)
	}
	defer tx.Rollback()
	q := cdb.New(tx)
	if userID == "" {
		newUserID, err := createUser(ctx, q, now, r)
		if err != nil {
			return err
		}
		userID = newUserID
	}
	if userID == "" {
		return srvu.Err(http.StatusBadRequest, fmt.Errorf("missing userID"))
	}
	invite, err := q.DeleteInvite(ctx, cdb.DeleteInviteParams{ID: inviteID, ExpiresAt: now.UnixMilli()})
	if err != nil || invite.ID == "" {
		return srvu.Err(http.StatusNotFound, fmt.Errorf("invalid invite: %s", inviteID))
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
}

func createUser(ctx context.Context, q *cdb.Queries, now time.Time, r *http.Request) (string, error) {
	username := r.FormValue("username")
	password := r.FormValue("password")
	if username == "" || password == "" {
		return "", srvu.Err(http.StatusBadRequest, fmt.Errorf("missing username or password"))
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", srvu.Err(http.StatusInternalServerError, fmt.Errorf("hashing password: %w", err))
	}
	user, err := q.CreateUser(ctx, cdb.CreateUserParams{
		ID:        NewId(),
		CreatedAt: now.UnixMilli(),
		UpdatedAt: now.UnixMilli(),
	})
	if err != nil {
		return "", srvu.Err(http.StatusInternalServerError, fmt.Errorf("create user: %w", err))
	}
	if err := q.CreatePasswordAuth(ctx, cdb.CreatePasswordAuthParams{
		UserID:   user.ID,
		Username: username,
		Hash:     string(hash),
	}); err != nil {
		return "", srvu.Err(http.StatusInternalServerError, fmt.Errorf("add user password: %w", err))
	}
	return user.ID, nil
}
