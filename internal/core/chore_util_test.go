package core_test

import (
	"context"
	"database/sql"
	"fmt"
	choretracker "github.com/SimonSchneider/chore-tracker"
	"github.com/SimonSchneider/chore-tracker/internal/cdb"
	"github.com/SimonSchneider/chore-tracker/internal/core"
	"github.com/SimonSchneider/chore-tracker/pkg/auth"
	"github.com/SimonSchneider/goslu/srvu"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"time"
)

func Panic(err error) {
	if err != nil {
		panic(err)
	}
}

func Must[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}
	return t
}

func Must2[T any, U any](t T, u U, err error) (T, U) {
	if err != nil {
		panic(err)
	}
	return t, u
}

type TestTemplateProvider struct {
	exec map[string]any
}

func (t *TestTemplateProvider) Lookup(name string) *template.Template {
	return nil
}

func (t *TestTemplateProvider) ExecuteTemplate(w io.Writer, name string, data interface{}) error {
	t.exec[name] = data
	return nil
}

func GetTpl[T any](t *TestTemplateProvider, name string) T {
	return t.exec[name].(T)
}

func Setup() (context.Context, *Client, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	db, err := core.GetMigratedDB(ctx, choretracker.StaticEmbeddedFS, "static/migrations", ":memory:")
	if err != nil {
		panic(fmt.Sprintf("failed to create test db: %s", err))
	}
	ctx = srvu.ContextWithLogger(ctx, srvu.LogToOutput(log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)))
	tplProv := &TestTemplateProvider{exec: make(map[string]any)}
	view := core.NewView(tplProv)
	tokenStore := auth.NewInMemoryTokenStore()
	authCfg := auth.Config{
		Provider:                    core.NewAuthProvider(db),
		UnauthorizedRedirect:        "/login",
		DefaultLogoutRedirect:       "/login",
		DefaultLoginSuccessRedirect: "/",
		LoginFailedRedirect:         "/login",
		RedirectParam:               "redirect",
		SessionsPath:                "/sessions/",
		CSRFTokenFieldName:          "CSRFToken",
		SessionCookie: auth.CookieConfig{
			Name:        "session",
			Expire:      1 * time.Hour,
			TokenLength: 32,
			Store:       tokenStore,
		},
		RefreshCookie: auth.CookieConfig{},
	}
	mux := core.Mux(db, view, authCfg)
	client := &Client{db: db, mux: mux, tokenStore: tokenStore, tmpl: tplProv, authCookieName: authCfg.SessionCookie.Name}
	return ctx, client, cancel
}

type Client struct {
	authCookieName string
	db             *sql.DB
	mux            http.Handler
	tokenStore     auth.SessionStore
	tmpl           *TestTemplateProvider
}

func (c *Client) DBQuery() *cdb.Queries {
	return cdb.New(c.db)
}

type ClientToken struct {
	cookieName string
	val        string
	CSRF       string
}

func (t ClientToken) Auth(r *http.Request) *http.Request {
	r.AddCookie(&http.Cookie{Name: t.cookieName, Value: t.val})
	return r
}

func (c *Client) NewToken(ctx context.Context) (*ClientToken, error) {
	u, err := c.DBQuery().CreateUser(ctx, cdb.CreateUserParams{ID: "test"})
	if err != nil {
		return nil, err
	}
	token := core.NewId()
	csrfToken := core.NewId()
	if err := c.tokenStore.StoreSession(ctx, auth.Session{
		UserID:    u.ID,
		Token:     token,
		CSRFToken: csrfToken,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}); err != nil {
		return nil, err
	}
	return &ClientToken{cookieName: c.authCookieName, val: token, CSRF: csrfToken}, nil
}

func (c *Client) Serve(r *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c.mux.ServeHTTP(w, r)
	return w
}

func NewFormReq(ctx context.Context, method, uri string, vals map[string]string) *http.Request {
	urlVals := &url.Values{}
	for k, v := range vals {
		urlVals.Set(k, v)
	}
	r := httptest.NewRequestWithContext(ctx, method, uri, strings.NewReader(urlVals.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return r
}

type ChoreReq struct {
	ctx    context.Context
	client *Client
	token  *ClientToken
	req    *http.Request
}

func NewChoreReq(ctx context.Context, client *Client) *ChoreReq {
	return &ChoreReq{ctx: ctx, client: client}
}

func (r *ChoreReq) Method(method string, uri string, body io.Reader) *ChoreReq {
	r.req = httptest.NewRequestWithContext(r.ctx, method, uri, body)
	return r
}

func (r *ChoreReq) Form(method, uri, csrf string, form map[string]string) *ChoreReq {
	if form == nil {
		form = make(map[string]string)
	}
	form["CSRFToken"] = csrf
	r.req = NewFormReq(r.ctx, method, uri, form)
	return r
}

func (r *ChoreReq) Get(uri string) *ChoreReq {
	r.req = httptest.NewRequestWithContext(r.ctx, http.MethodGet, uri, nil)
	return r
}

func (r *ChoreReq) Auth(t *ClientToken) *ChoreReq {
	r.token = t
	return r
}

func (r *ChoreReq) Do() *http.Response {
	if r.token != nil {
		r.req.AddCookie(&http.Cookie{Name: r.client.authCookieName, Value: r.token.val})
	}
	return r.client.Serve(r.req).Result()
}

func (r *ChoreReq) DoAndExp(status int) (*http.Response, error) {
	res := r.Do()
	if res.StatusCode != status {
		return nil, fmt.Errorf("unexpected status: %d (%+v)", res.StatusCode, res)
	}
	return res, nil
}

func (r *ChoreReq) DoAndFollow(status int) (*http.Response, error) {
	res := r.Do()
	if res.StatusCode != status {
		return nil, fmt.Errorf("failed to follow: %d (%+v)", res.StatusCode, res)
	}
	loc := res.Header.Get("Location")
	if loc != "" {
		return NewChoreReq(r.ctx, r.client).Auth(r.token).Get(res.Header.Get("Location")).DoAndFollow(http.StatusOK)
	}
	return res, nil
}

func NewChoreList(ctx context.Context, client *Client, token *ClientToken, formVals map[string]string) (*core.ChoreListView, error) {
	_, err := NewChoreReq(ctx, client).Auth(token).Form("POST", "/chore-lists/", token.CSRF, formVals).DoAndFollow(http.StatusSeeOther)
	if err != nil {
		return nil, err
	}
	cl := GetTpl[core.ChoreListView](client.tmpl, "chore_list.page.gohtml")
	if cl.List.ID == "" {
		return nil, fmt.Errorf("chore list id is empty")
	}
	return &cl, nil
}

func findInSlice[T any](slice []T, pred func(T) bool) *T {
	for i, v := range slice {
		if pred(v) {
			return &slice[i]
		}
	}
	return nil
}

func NewChore(ctx context.Context, client *Client, token *ClientToken, formVals map[string]string) (*core.Chore, error) {
	if _, err := NewChoreReq(ctx, client).Auth(token).Form("POST", "/chores/", token.CSRF, formVals).DoAndFollow(http.StatusSeeOther); err != nil {
		return nil, err
	}
	cl := GetTpl[core.ChoreListView](client.tmpl, "chore_list.page.gohtml")
	if cl.Chores == nil {
		return nil, fmt.Errorf("chore not found")
	}
	c := findInSlice(cl.Chores.Chores, func(c core.Chore) bool {
		return c.Name == formVals["name"]
	})
	return c, nil
}

func GetChore(ctx context.Context, client *Client, token *ClientToken, choreListID, choreID string) (*core.Chore, error) {
	if _, err := NewChoreReq(ctx, client).Auth(token).Get(fmt.Sprintf("/chore-lists/%s", choreListID)).DoAndFollow(http.StatusOK); err != nil {
		return nil, err
	}
	cl := GetTpl[core.ChoreListView](client.tmpl, "chore_list.page.gohtml")
	if cl.Chores == nil {
		return nil, fmt.Errorf("chore not found")
	}
	c := findInSlice(cl.Chores.Chores, func(c core.Chore) bool {
		return c.ID == choreID
	})
	return c, nil
}
