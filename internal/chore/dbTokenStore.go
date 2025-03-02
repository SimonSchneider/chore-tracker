package chore

import (
	"context"
	"database/sql"
	"errors"
	"github.com/SimonSchneider/chore-tracker/internal/cdb"
	"time"
)

type DBTokenStore struct {
	DB *sql.DB
}

func (s *DBTokenStore) StoreToken(ctx context.Context, userID, token string, expiresAt time.Time) error {
	return cdb.New(s.DB).CreateToken(ctx, cdb.CreateTokenParams{UserID: userID, Token: token, ExpiresAt: expiresAt.UnixMilli()})
}

func (s *DBTokenStore) DeleteTokens(ctx context.Context, userID string) error {
	return cdb.New(s.DB).DeleteTokensByUserId(ctx, userID)
}

func (s *DBTokenStore) VerifyToken(ctx context.Context, token string, now time.Time) (string, bool, error) {
	res, err := cdb.New(s.DB).GetToken(ctx, cdb.GetTokenParams{Token: token, ExpiresAt: now.UnixMilli()})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false, nil
		}
		return "", false, err
	}
	return res.UserID, true, nil
}
