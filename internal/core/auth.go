package core

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/SimonSchneider/chore-tracker/internal/cdb"
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

func (s *DBTokenStore) StoreToken(ctx context.Context, userID, token string, expiresAt time.Time) error {
	return cdb.New(s.DB).CreateToken(ctx, cdb.CreateTokenParams{UserID: userID, Token: token, ExpiresAt: expiresAt.UnixMilli()})
}

func (s *DBTokenStore) DeleteTokens(ctx context.Context, userID string) error {
	return cdb.New(s.DB).DeleteTokensByUserId(ctx, userID)
}

func (s *DBTokenStore) VerifyToken(ctx context.Context, token string, now time.Time) (string, time.Time, bool, error) {
	res, err := cdb.New(s.DB).GetToken(ctx, cdb.GetTokenParams{Token: token, ExpiresAt: now.UnixMilli()})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", time.Time{}, false, nil
		}
		return "", time.Time{}, false, err
	}
	return res.UserID, time.UnixMilli(res.ExpiresAt), true, nil
}
