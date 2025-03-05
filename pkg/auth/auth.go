package auth

import (
	"context"
	"fmt"
	"github.com/SimonSchneider/goslu/sid"
	"github.com/SimonSchneider/goslu/srvu"
	"net/http"
	"time"
)

/*
- not authed -> redirect to login page with redirect URL set to current page
- authedMW -> check cookie and either resolve or redirect to login path set userID in context
- in app - loginPage -> display login form which uses POST /login with userID and password and remember me
-x POST login -> check userID and password, if correct set session cookie and redirect to redirect url provided or default
-x POST logout -> clear session cookie and redirect to default

- need to support invites and generating a password for invited users
*/

type Provider interface {
	AuthenticateUser(ctx context.Context, r *http.Request) (userID string, err error)
	RenderLoginPage(ctx context.Context, w http.ResponseWriter, r *http.Request) error
}

type TokenStore interface {
	StoreToken(ctx context.Context, userID, token string, expiresAt time.Time) error
	DeleteTokens(ctx context.Context, userID string) error
	VerifyToken(ctx context.Context, token string, now time.Time) (string, bool, error)
}

type CookieConfig struct {
	Name        string
	Expire      time.Duration
	TokenLength int
	Store       TokenStore
}

type Config struct {
	Provider                    Provider
	UnauthorizedRedirect        string
	DefaultLogoutRedirect       string
	DefaultLoginSuccessRedirect string
	LoginFailedRedirect         string
	ShortLivedCookie            CookieConfig
	LongLivedCookie             CookieConfig
}

func (c *Config) LoginPage() http.Handler {
	return c.Middleware(true)(srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		existing, _ := GetUserID(ctx)
		if existing != "" {
			redirectUrl := r.URL.Query().Get("redirect")
			if redirectUrl == "" {
				redirectUrl = c.DefaultLoginSuccessRedirect
			}
			http.Redirect(w, r, redirectUrl, http.StatusSeeOther)
			return nil
		}
		return c.Provider.RenderLoginPage(ctx, w, r)
	}))
}

func (c *Config) LoginHandler() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		rememberMe := r.FormValue("rememberMe") == "on"
		redirectUrl := r.URL.Query().Get("redirect")
		if redirectUrl == "" {
			redirectUrl = c.DefaultLoginSuccessRedirect
		}
		userID, err := c.Provider.AuthenticateUser(ctx, r)
		if err != nil {
			srvu.GetLogger(ctx).Printf("failed to authenticate user: %v", err)
		}
		if userID == "" {
			http.Redirect(w, r, c.LoginFailedRedirect, http.StatusSeeOther)
			return nil
		}
		if rememberMe {
			if err := c.LongLivedCookie.generateStoreAndSetTokenCookie(ctx, userID, w); err != nil {
				return srvu.Err(http.StatusInternalServerError, err)
			}
		}
		if err := c.ShortLivedCookie.generateStoreAndSetTokenCookie(ctx, userID, w); err != nil {
			return srvu.Err(http.StatusInternalServerError, err)
		}
		http.Redirect(w, r, redirectUrl, http.StatusSeeOther)
		return nil
	})
}

func (c *Config) LogoutHandler() http.Handler {
	return c.Middleware(true)(srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		c.ShortLivedCookie.deleteCookie(w)
		c.LongLivedCookie.deleteCookie(w)
		if userID, err := GetUserID(ctx); err == nil {
			c.ShortLivedCookie.Store.DeleteTokens(ctx, userID)
			c.LongLivedCookie.Store.DeleteTokens(ctx, userID)
		}
		redirectUrl := r.URL.Query().Get("redirect")
		if redirectUrl == "" {
			redirectUrl = c.DefaultLogoutRedirect
		}
		http.Redirect(w, r, redirectUrl, http.StatusSeeOther)
		return nil
	}))
}

func (c *Config) Middleware(allowUnauthenticated bool) srvu.Middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			alreadyAuthed, err := GetUserID(r.Context())
			if err == nil && alreadyAuthed != "" {
				h.ServeHTTP(w, r)
				return
			}
			userID, err := c.ShortLivedCookie.verifyToken(r)
			if err != nil {
				userID, err = c.LongLivedCookie.verifyToken(r)
				if err != nil {
					if allowUnauthenticated {
						ctx := setUserID(r.Context(), "")
						h.ServeHTTP(w, r.WithContext(ctx))
						return
					}
					c.ShortLivedCookie.deleteCookie(w)
					c.LongLivedCookie.deleteCookie(w)
					http.Redirect(w, r, c.UnauthorizedRedirect, http.StatusSeeOther)
					return
				}
				if err := c.ShortLivedCookie.generateStoreAndSetTokenCookie(r.Context(), userID, w); err != nil {
					http.Redirect(w, r, c.UnauthorizedRedirect, http.StatusSeeOther)
					return
				}
				if err := c.LongLivedCookie.generateStoreAndSetTokenCookie(r.Context(), userID, w); err != nil {
					http.Redirect(w, r, c.UnauthorizedRedirect, http.StatusSeeOther)
					return
				}
			}
			ctx := setUserID(r.Context(), userID)
			h.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func (c *CookieConfig) generateStoreAndSetTokenCookie(ctx context.Context, userID string, w http.ResponseWriter) error {
	token, exp, err := generateToken(c.Expire, c.TokenLength)
	if err != nil {
		return err
	}
	if err := c.Store.StoreToken(ctx, userID, token, exp); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     c.Name,
		Value:    token,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(c.Expire.Seconds()),
	})
	return nil
}

func (c *CookieConfig) verifyToken(r *http.Request) (string, error) {
	cookie, err := r.Cookie(c.Name)
	if err != nil || cookie == nil {
		return "", fmt.Errorf("failed to get cookie from cookie: %w", err)
	}
	if err := cookie.Valid(); err != nil {
		return "", fmt.Errorf("invalid cookie: %w", err)
	}
	token := cookie.Value
	userID, ok, err := c.Store.VerifyToken(r.Context(), token, time.Now())
	if err != nil {
		return "", fmt.Errorf("failed to verify cookie: %w", err)
	}
	if !ok {
		return "", fmt.Errorf("invalid cookie")
	}
	return userID, nil
}

func (c *CookieConfig) deleteCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     c.Name,
		Value:    "",
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
		MaxAge:   -1,
	})
}

func generateToken(expire time.Duration, tokenLength int) (string, time.Time, error) {
	sessionToken, err := sid.NewString(tokenLength)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to generate session token: %w", err)
	}
	expiresAt := time.Now().Add(expire)
	return sessionToken, expiresAt, nil
}

var userIDContextKey = struct{}{}

func GetUserID(ctx context.Context) (string, error) {
	userID, ok := ctx.Value(userIDContextKey).(string)
	if !ok || userID == "" {
		return "", fmt.Errorf("failed to get userID from context")
	}
	return userID, nil
}

func MustGetUserID(ctx context.Context) string {
	userID, err := GetUserID(ctx)
	if err != nil {
		panic(err)
	}
	return userID
}

func setUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDContextKey, userID)
}
