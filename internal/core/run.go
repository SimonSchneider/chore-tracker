package core

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	choretracker "github.com/SimonSchneider/chore-tracker"
	"github.com/SimonSchneider/chore-tracker/internal/cdb"
	"github.com/SimonSchneider/chore-tracker/pkg/auth"
	"github.com/SimonSchneider/chore-tracker/pkg/httpu"
	"github.com/SimonSchneider/goslu/config"
	"github.com/SimonSchneider/goslu/migrate"
	"github.com/SimonSchneider/goslu/srvu"
	"github.com/SimonSchneider/goslu/templ"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func LoginPage(view *View) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		session, _ := auth.GetSession(ctx)
		if session.UserID != "" {
			http.Redirect(w, r, "/", http.StatusFound)
			return nil
		}
		return view.LoginPage(w, r)
	})
}

func Mux(db *sql.DB, view *View, authConfig auth.Config, apiKey string) http.Handler {
	inviteStore := &InviteStore{db: db, view: view}
	mux := http.NewServeMux()
	mux.Handle("GET /login", srvu.With(LoginPage(view), authConfig.Middleware(true, true)))
	mux.Handle("POST /logout", authConfig.DeleteSessionHandler())
	mux.Handle(authConfig.SessionsPath, authConfig.SessionHandler())
	mux.Handle("GET /settings", srvu.With(SettingsPage(view, db), authConfig.Middleware(false, false)))

	httpu.HandleNested(mux, "/invites/", auth.InviteHandler(inviteStore, authConfig))
	mux.Handle("/chore-lists/", srvu.With(ChoreListMux(db, view, inviteStore), authConfig.Middleware(false, false)))
	mux.Handle("/api/chore-lists/", srvu.With(ChoreListAPIMux(db, view, apiKey)))
	mux.Handle("/chores/", srvu.With(ChoreMux(db, view), authConfig.Middleware(false, false)))
	mux.Handle("/{$}", http.RedirectHandler("/chore-lists/", http.StatusFound))
	return mux
}

func Run(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, getEnv func(string) string, getwd func() (string, error)) error {
	cfg, err := parseConfig(args[1:], getEnv)
	if err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}
	public, tmplProv, err := templ.GetPublicAndTemplates(choretracker.StaticEmbeddedFS, &templ.Config{
		Watch:        cfg.Watch,
		TmplPatterns: []string{"templates/*.gohtml", "templates/*.goics"},
	})
	if err != nil {
		return fmt.Errorf("sub static: %w", err)
	}
	return RunCfg(ctx, stdout, cfg, public, tmplProv)
}

func RunCfg(ctx context.Context, stdout io.Writer, cfg Config, public fs.FS, tmplProv templ.TemplateProvider) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, os.Kill)
	defer cancel()
	logger := srvu.LogToOutput(log.New(stdout, "", log.LstdFlags|log.Lshortfile))

	db, err := GetMigratedDB(ctx, choretracker.StaticEmbeddedFS, "static/migrations", cfg.DbURL)
	if err != nil {
		return fmt.Errorf("failed to migrate db: %w", err)
	}

	view := NewView(tmplProv)
	authConfig := auth.Config{
		Provider:                    &AuthProvider{db: db},
		RedirectParam:               "redirect",
		UnauthorizedRedirect:        "/login",
		DefaultLogoutRedirect:       "/login",
		LoginFailedRedirect:         "/login",
		DefaultLoginSuccessRedirect: "/",
		SessionsPath:                "/sessions/",
		CSRFTokenFieldName:          "CSRFToken",
		SessionCookie: auth.CookieConfig{
			Name:          "chore_session",
			Expire:        30 * time.Minute,
			RefreshMargin: 5 * time.Minute,
			TokenLength:   32,
			Store:         auth.NewInMemoryTokenStore(),
		},
		RefreshCookie: auth.CookieConfig{
			Name:          "chore_refresh_session",
			Expire:        24 * time.Hour * 30,
			RefreshMargin: 24 * time.Hour,
			TokenLength:   102,
			Store:         &DBTokenStore{DB: db},
		},
	}

	mux := http.NewServeMux()
	httpu.HandleNested(mux, "GET /static/public/", srvu.With(http.FileServerFS(public), srvu.WithCacheCtrlHeader(365*24*time.Hour)))
	mux.Handle("/", Mux(db, view, authConfig, cfg.ApiKey))

	srv := &http.Server{
		BaseContext: func(listener net.Listener) context.Context {
			return ctx
		},
		Addr:    cfg.Addr,
		Handler: srvu.With(mux, srvu.WithCompression(), srvu.WithLogger(logger)),
	}
	logger.Printf("starting chore server, listening on %s\n  sqliteDB: %s", cfg.Addr, cfg.DbURL)
	if cfg.GenInv {
		invID, err := GenerateInvite(ctx, db)
		if err != nil {
			return fmt.Errorf("failed to generate invite: %w", err)
		}
		logger.Printf("created invite: http://localhost%s/invites/%s", cfg.Addr, invID)
	}
	return srvu.RunServerGracefully(ctx, srv, logger)
}

const systemUserID = "system"

func GenerateInvite(ctx context.Context, db *sql.DB) (string, error) {
	q := cdb.New(db)
	systemUser, err := q.GetUser(ctx, systemUserID)
	if err != nil || systemUser.ID == "" {
		systemUser, err = cdb.New(db).CreateUser(ctx, cdb.CreateUserParams{
			ID:          systemUserID,
			DisplayName: systemUserID,
			CreatedAt:   time.Now().UnixMilli(),
			UpdatedAt:   time.Now().UnixMilli(),
		})
		if err != nil {
			return "", fmt.Errorf("create system user: %w", err)
		}
	}
	inv, err := cdb.New(db).CreateInvite(ctx, cdb.CreateInviteParams{
		ID:          NewId(),
		CreatedAt:   time.Now().UnixMilli(),
		ExpiresAt:   time.Now().Add(5 * time.Hour).UnixMilli(),
		ChoreListID: sql.NullString{},
		CreatedBy:   systemUser.ID,
	})
	if err != nil {
		return "", fmt.Errorf("create invite: %w", err)
	}
	return inv.ID, nil
}

type Config struct {
	Addr   string
	Watch  bool
	DbURL  string
	GenInv bool
	ApiKey string
}

func parseConfig(args []string, getEnv func(string) string) (cfg Config, err error) {
	err = config.ParseInto(&cfg, flag.NewFlagSet("", flag.ExitOnError), args, getEnv)
	return cfg, err
}

func GetMigratedDB(ctx context.Context, dir fs.FS, path string, conn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", conn)
	if err != nil {
		return nil, fmt.Errorf("opening db: %w", err)
	}
	migrations, err := fs.Sub(dir, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get migrations: %w", err)
	}
	if err := migrate.Migrate(ctx, migrations, db); err != nil {
		return nil, fmt.Errorf("failed to migrate db: %w", err)
	}
	return db, nil
}
