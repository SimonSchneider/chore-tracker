package chore

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/SimonSchneider/chore-tracker/internal/cdb"
	"github.com/SimonSchneider/goslu/templ"
	"golang.org/x/crypto/bcrypt"
	"net/http"
)

type AuthProvider struct {
	db    *sql.DB
	tmpls templ.TemplateProvider
}

func (a *AuthProvider) AuthenticateUser(ctx context.Context, r *http.Request) (string, error) {
	username := r.FormValue("username")
	password := r.FormValue("password")
	if username == "" || password == "" {
		return "", fmt.Errorf("missing username or password")
	}
	fmt.Printf("Authenticating pwAuth %s ...\n", username)
	pwAuth, err := cdb.New(a.db).GetPasswordAuthByUsername(ctx, username)
	if err != nil {
		return "", fmt.Errorf("getting pwAuth: %w", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(pwAuth.Hash), []byte(password)); err != nil {
		return "", fmt.Errorf("invalid password")
	}
	return pwAuth.UserID, nil
}

func (a *AuthProvider) RenderLoginPage(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	return a.tmpls.ExecuteTemplate(w, "login.gohtml", nil)
}
