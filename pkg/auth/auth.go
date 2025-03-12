package auth

import (
	"context"
	"fmt"
	"github.com/SimonSchneider/goslu/sid"
	"github.com/SimonSchneider/goslu/srvu"
	"net/http"
	"path"
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
}

type TokenStore interface {
	StoreToken(ctx context.Context, userID, token string, expiresAt time.Time) error
	DeleteTokens(ctx context.Context, userID string) error
	VerifyToken(ctx context.Context, token string, now time.Time) (string, time.Time, bool, error)
}

type CookieConfig struct {
	Name          string
	Expire        time.Duration
	RefreshMargin time.Duration
	TokenLength   int
	Store         TokenStore
}

type Config struct {
	Provider                    Provider
	RedirectParam               string
	UnauthorizedRedirect        string
	DefaultLogoutRedirect       string
	DefaultLoginSuccessRedirect string
	LoginFailedRedirect         string
	SessionsPath                string
	SessionCookie               CookieConfig
	RefreshCookie               CookieConfig
}

func (c *Config) SessionHandler() http.Handler {
	mux := http.NewServeMux()
	prefix := c.sessionPathPrefix()
	mux.Handle(path.Join(prefix, "/refresh"), c.RefreshHandler())
	mux.Handle(fmt.Sprintf("POST %s/{$}", path.Join(prefix, "/")), c.CreateSessionHandler())
	mux.Handle(fmt.Sprintf("DELETE %s/{$}", path.Join(prefix, "/")), c.DeleteSessionHandler())
	return mux
}

func (c *Config) sessionPathPrefix() string {
	if c.SessionsPath == "" {
		panic("SessionsPath: must specify sessions path")
	}
	return c.SessionsPath
}

func (c *Config) RefreshHandler() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		userID, refresh, err := c.RefreshCookie.verifyToken(r)
		redirect := r.URL.Query().Get(c.redirectParam())
		if redirect == "" {
			redirect = c.DefaultLoginSuccessRedirect
		}
		unauthorizedRedirect := func() {
			http.Redirect(w, r, fmt.Sprintf("%s?%s=%s", c.UnauthorizedRedirect, c.redirectParam(), redirect), http.StatusSeeOther)
		}
		if err != nil {
			c.SessionCookie.deleteCookie(w, c.sessionCookiePath())
			c.RefreshCookie.deleteCookie(w, c.refreshCookiePath())
			unauthorizedRedirect()
			return nil
		}
		if err := c.SessionCookie.generateStoreAndSetTokenCookie(r.Context(), userID, c.sessionCookiePath(), w); err != nil {
			srvu.GetLogger(r.Context()).Printf("failed generate short token: %v", err)
			unauthorizedRedirect()
			return nil
		}
		if refresh {
			if err := c.RefreshCookie.generateStoreAndSetTokenCookie(r.Context(), userID, c.refreshCookiePath(), w); err != nil {
				srvu.GetLogger(r.Context()).Printf("failed generate long token: %v", err)
			}
		}
		http.Redirect(w, r, redirect, http.StatusSeeOther)
		return nil
	})
}

func (c *Config) refreshCookiePath() string {
	return path.Join(c.SessionsPath, "/refresh/")
}

func (c *Config) sessionCookiePath() string {
	return "/"
}

func (c *Config) redirectParam() string {
	if c.RedirectParam == "" {
		panic("redirect param is required")
	}
	return c.RedirectParam
}

func (c *Config) CreateSessionHandler() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		rememberMe := r.FormValue("rememberMe") == "on"
		redirectUrl := r.URL.Query().Get(c.redirectParam())
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
			fmt.Printf("remember me is on: %s, %s - %+v\n", userID, c.refreshCookiePath(), r.Form)
			if err := c.RefreshCookie.generateStoreAndSetTokenCookie(ctx, userID, c.refreshCookiePath(), w); err != nil {
				return srvu.Err(http.StatusInternalServerError, err)
			}
		} else {
			c.RefreshCookie.deleteCookie(w, c.refreshCookiePath())
		}
		if err := c.SessionCookie.generateStoreAndSetTokenCookie(ctx, userID, "/", w); err != nil {
			return srvu.Err(http.StatusInternalServerError, err)
		}
		http.Redirect(w, r, redirectUrl, http.StatusSeeOther)
		return nil
	})
}

func (c *Config) DeleteSessionHandler() http.Handler {
	return c.Middleware(true, false)(srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		c.SessionCookie.deleteCookie(w, c.sessionCookiePath())
		c.RefreshCookie.deleteCookie(w, c.refreshCookiePath())
		if userID, err := GetUserID(ctx); err == nil {
			c.SessionCookie.Store.DeleteTokens(ctx, userID)
			c.RefreshCookie.Store.DeleteTokens(ctx, userID)
		}
		redirectUrl := r.URL.Query().Get(c.redirectParam())
		if redirectUrl == "" {
			redirectUrl = c.DefaultLogoutRedirect
		}
		http.Redirect(w, r, redirectUrl, http.StatusSeeOther)
		return nil
	}))
}

func (c *Config) refreshPath() string {
	return fmt.Sprintf("%s/refresh", c.SessionsPath)
}

func (c *Config) Middleware(allowUnauthenticated, followRedirect bool) srvu.Middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			alreadyAuthed, err := GetUserID(r.Context())
			redirectUrl := r.URL.Query().Get(c.redirectParam())
			if err == nil && alreadyAuthed != "" {
				if followRedirect && redirectUrl != "" {
					http.Redirect(w, r, redirectUrl, http.StatusSeeOther)
					return
				}
				h.ServeHTTP(w, r)
				return
			}
			userID, refresh, err := c.SessionCookie.verifyToken(r)
			if err != nil && allowUnauthenticated {
				h.ServeHTTP(w, r.WithContext(setUserID(r.Context(), "")))
			} else if err != nil {
				http.Redirect(w, r, fmt.Sprintf("%s?%s=%s", c.refreshPath(), c.redirectParam(), r.URL.Path), http.StatusSeeOther)
			} else {
				if refresh {
					if err := c.SessionCookie.generateStoreAndSetTokenCookie(r.Context(), userID, c.refreshCookiePath(), w); err != nil {
						srvu.GetLogger(r.Context()).Printf("failed to refresh session cookie: %v", err)
					}
				}
				if followRedirect && redirectUrl != "" {
					http.Redirect(w, r, redirectUrl, http.StatusSeeOther)
					return
				}
				h.ServeHTTP(w, r.WithContext(setUserID(r.Context(), userID)))
			}
		})
	}
}

func (c *CookieConfig) generateStoreAndSetTokenCookie(ctx context.Context, userID, path string, w http.ResponseWriter) error {
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
		Path:     path,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(c.Expire.Seconds()),
	})
	return nil
}

func timeMin(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func (c *CookieConfig) verifyToken(r *http.Request) (string, bool, error) {
	cookie, err := r.Cookie(c.Name)
	if err != nil || cookie == nil {
		return "", false, fmt.Errorf("failed to get cookie from cookie: %w", err)
	}
	if err := cookie.Valid(); err != nil {
		return "", false, fmt.Errorf("invalid cookie: %w", err)
	}
	token := cookie.Value
	userID, exp, ok, err := c.Store.VerifyToken(r.Context(), token, time.Now())
	if err != nil {
		return "", false, fmt.Errorf("failed to verify cookie: %w", err)
	}
	if !ok {
		return "", false, fmt.Errorf("invalid cookie")
	}
	exp = timeMin(cookie.Expires, exp)
	return userID, time.Now().Add(c.RefreshMargin).After(exp), nil
}

func (c *CookieConfig) deleteCookie(w http.ResponseWriter, path string) {
	http.SetCookie(w, &http.Cookie{
		Name:     c.Name,
		Value:    "",
		Path:     path,
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
