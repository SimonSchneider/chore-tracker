package core

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/SimonSchneider/chore-tracker/internal/cdb"
	"github.com/SimonSchneider/chore-tracker/pkg/auth"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"time"
)

type AuthProvider struct {
	db *sql.DB
}

func NewAuthProvider(db *sql.DB) *AuthProvider {
	return &AuthProvider{db: db}
}

func (a *AuthProvider) AuthenticateUser(ctx context.Context, r *http.Request) (string, error) {
	username := r.FormValue("username")
	password := r.FormValue("password")
	if username == "" || password == "" {
		return "", fmt.Errorf("missing username or password")
	}
	pwAuth, err := cdb.New(a.db).GetPasswordAuthByUsername(ctx, username)
	if err != nil {
		return "", fmt.Errorf("getting pwAuth: %w", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(pwAuth.Hash), []byte(password)); err != nil {
		return "", fmt.Errorf("invalid password")
	}
	return pwAuth.UserID, nil
}

type DBTokenStore struct {
	DB *sql.DB
}

func (s *DBTokenStore) VerifyCSRFToken(ctx context.Context, userID, csrfToken string, now time.Time) (bool, error) {
	res, err := cdb.New(s.DB).GetCsrfTokenByUserAndCsrfToken(ctx, cdb.GetCsrfTokenByUserAndCsrfTokenParams{UserID: userID, CsrfToken: csrfToken, ExpiresAt: now.UnixMilli()})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return res == 1, nil
}

func sessionCreateParams(session auth.Session) cdb.CreateTokenParams {
	return cdb.CreateTokenParams{UserID: session.UserID, Token: session.Token, CsrfToken: session.CSRFToken, ExpiresAt: session.ExpiresAt.UnixMilli()}
}

func (s *DBTokenStore) StoreSession(ctx context.Context, session auth.Session) error {
	return cdb.New(s.DB).CreateToken(ctx, sessionCreateParams(session))
}

func (s *DBTokenStore) ReplaceSession(ctx context.Context, oldSession, newSession auth.Session) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("unable to start tx: %w", err)
	}
	defer tx.Rollback()
	q := cdb.New(tx)
	if err := q.DeleteToken(ctx, oldSession.Token); err != nil {
		return fmt.Errorf("deleting old session: %w", err)
	}
	if err := q.CreateToken(ctx, sessionCreateParams(newSession)); err != nil {
		return fmt.Errorf("creating new session: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing tx: %w", err)
	}
	return nil
}

func (s *DBTokenStore) DeleteSessions(ctx context.Context, userID string) error {
	return cdb.New(s.DB).DeleteTokensByUserId(ctx, userID)
}

func (s *DBTokenStore) VerifySession(ctx context.Context, token string, now time.Time) (auth.Session, bool, error) {
	res, err := cdb.New(s.DB).GetToken(ctx, cdb.GetTokenParams{Token: token, ExpiresAt: now.UnixMilli()})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return auth.Session{}, false, nil
		}
		return auth.Session{}, false, err
	}
	return auth.Session{
		UserID:    res.UserID,
		Token:     res.Token,
		CSRFToken: res.CsrfToken,
		ExpiresAt: time.UnixMilli(res.ExpiresAt),
	}, true, nil
}

func (s *DBTokenStore) GC(ctx context.Context, now time.Time) error {
	return cdb.New(s.DB).DeleteExpiredTokens(ctx, now.UnixMilli())
}
