package auth

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"time"

	"github.com/SimonSchneider/goslu/sid"
	"github.com/SimonSchneider/goslu/srvu"
)

type Provider interface {
	AuthenticateUser(ctx context.Context, r *http.Request) (userID string, err error)
}

type Session struct {
	UserID    string
	Token     string
	ExpiresAt time.Time
}

type SessionStore interface {
	StoreSession(ctx context.Context, session Session) error
	DeleteSessions(ctx context.Context, userID string) error
	VerifySession(ctx context.Context, token string, now time.Time) (Session, bool, error)
}

type CookieConfig struct {
	Name          string
	Expire        time.Duration
	RefreshMargin time.Duration
	TokenLength   int
	Store         SessionStore
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
	return srvu.With(mux, func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Pragma", "no-cache")
			w.Header().Add("Cache-Control", `no-cache, no-store, must-revalidate, max-age=0`)
			handler.ServeHTTP(w, r)
		})
	})
}

func (c *Config) sessionPathPrefix() string {
	if c.SessionsPath == "" {
		panic("SessionsPath: must specify sessions path")
	}
	return c.SessionsPath
}

func (c *Config) RefreshHandler() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		session, refresh, err := c.RefreshCookie.verifyToken(r, time.Now())
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
		if err := c.SessionCookie.generateStoreAndSetSessionCookie(r.Context(), session.UserID, c.sessionCookiePath(), w); err != nil {
			srvu.GetLogger(r.Context()).Printf("failed generate short token: %v", err)
			unauthorizedRedirect()
			return nil
		}
		if refresh {
			if err := c.RefreshCookie.generateStoreAndSetSessionCookie(r.Context(), session.UserID, c.refreshCookiePath(), w); err != nil {
				srvu.GetLogger(r.Context()).Printf("failed generate long token: %v", err)
			}
		}
		http.Redirect(w, r, redirect, http.StatusTemporaryRedirect)
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

func (c *Config) getRedirectURL(r *http.Request, fallback string) string {
	red := r.URL.Query().Get(c.redirectParam())
	if red == "" {
		return fallback
	}
	return red
}

func (c *Config) CreateSessionHandler() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		rememberMe := r.FormValue("rememberMe") == "on"
		redirectUrl := c.getRedirectURL(r, c.DefaultLoginSuccessRedirect)
		userID, err := c.Provider.AuthenticateUser(ctx, r)
		if err != nil || userID == "" {
			http.Redirect(w, r, c.LoginFailedRedirect, http.StatusSeeOther)
			return nil
		}
		if rememberMe {
			if err := c.RefreshCookie.generateStoreAndSetSessionCookie(ctx, userID, c.refreshCookiePath(), w); err != nil {
				return srvu.Err(http.StatusInternalServerError, err)
			}
		} else {
			c.RefreshCookie.deleteCookie(w, c.refreshCookiePath())
		}
		if err := c.SessionCookie.generateStoreAndSetSessionCookie(ctx, userID, "/", w); err != nil {
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
		if session, err := GetSession(ctx); err == nil {
			c.SessionCookie.Store.DeleteSessions(ctx, session.UserID)
			c.RefreshCookie.Store.DeleteSessions(ctx, session.UserID)
		}
		redirectUrl := c.getRedirectURL(r, c.DefaultLogoutRedirect)
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
			existingSession, err := GetSession(r.Context())
			redirectUrl := c.getRedirectURL(r, "")
			if err == nil && existingSession.UserID != "" {
				if followRedirect && redirectUrl != "" {
					http.Redirect(w, r, redirectUrl, http.StatusSeeOther)
					return
				}
				h.ServeHTTP(w, r)
				return
			}
			now := time.Now()
			session, refresh, err := c.SessionCookie.verifyToken(r, now)
			if err != nil && allowUnauthenticated {
				h.ServeHTTP(w, r.WithContext(withoutSession(r.Context())))
			} else if err != nil {
				http.Redirect(w, r, fmt.Sprintf("%s?%s=%s", c.refreshPath(), c.redirectParam(), r.URL.RequestURI()), http.StatusTemporaryRedirect)
			} else {
				if refresh {
					if err := c.SessionCookie.generateStoreAndSetSessionCookie(r.Context(), session.UserID, c.sessionCookiePath(), w); err != nil {
						srvu.GetLogger(r.Context()).Printf("failed to refresh session cookie: %v", err)
					}
				}
				if followRedirect && redirectUrl != "" {
					http.Redirect(w, r, redirectUrl, http.StatusSeeOther)
					return
				}
				h.ServeHTTP(w, r.WithContext(withSession(r.Context(), session)))
			}
		})
	}
}

func (c *CookieConfig) generateStoreAndSetSessionCookie(ctx context.Context, userID, path string, w http.ResponseWriter) error {
	session, err := generateSession(userID, c.Expire, c.TokenLength)
	if err != nil {
		return err
	}
	if err := c.Store.StoreSession(ctx, session); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     c.Name,
		Value:    session.Token,
		HttpOnly: true,
		Path:     path,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
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

func (c *CookieConfig) verifyToken(r *http.Request, now time.Time) (Session, bool, error) {
	cookie, err := r.Cookie(c.Name)
	if err != nil || cookie == nil {
		return Session{}, false, fmt.Errorf("failed to get cookie from cookie: %w", err)
	}
	if err := cookie.Valid(); err != nil {
		return Session{}, false, fmt.Errorf("invalid cookie: %w", err)
	}
	token := cookie.Value
	session, ok, err := c.Store.VerifySession(r.Context(), token, now)
	if err != nil {
		return Session{}, false, fmt.Errorf("failed to verify cookie: %w", err)
	}
	if !ok {
		return Session{}, false, fmt.Errorf("invalid cookie")
	}
	return session, now.Add(c.RefreshMargin).After(session.ExpiresAt), nil
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

func generateSession(userID string, expire time.Duration, tokenLength int) (Session, error) {
	sessionToken, err := sid.NewString(tokenLength)
	if err != nil {
		return Session{}, fmt.Errorf("failed to generate session token: %w", err)
	}
	expiresAt := time.Now().Add(expire)
	return Session{
		Token:     sessionToken,
		ExpiresAt: expiresAt,
		UserID:    userID,
	}, nil
}

var sessionContextKey = struct{}{}

func GetSession(ctx context.Context) (Session, error) {
	session, ok := ctx.Value(sessionContextKey).(Session)
	if !ok || session.UserID == "" {
		return Session{}, fmt.Errorf("failed to get userID from context")
	}
	return session, nil
}

func MustGetSession(ctx context.Context) Session {
	session, err := GetSession(ctx)
	if err != nil {
		panic(err)
	}
	return session
}

func withSession(ctx context.Context, session Session) context.Context {
	return context.WithValue(ctx, sessionContextKey, session)
}

func withoutSession(ctx context.Context) context.Context {
	return context.WithValue(ctx, sessionContextKey, nil)
}
