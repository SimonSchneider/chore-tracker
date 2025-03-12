package core

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	choretracker "github.com/SimonSchneider/chore-tracker"
	"github.com/SimonSchneider/chore-tracker/internal/cdb"
	"github.com/SimonSchneider/chore-tracker/pkg/auth"
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
		return view.LoginPage(w, r)
	})
}

func Mux(db *sql.DB, view *View, authConfig auth.Config) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /login", srvu.With(LoginPage(view), authConfig.Middleware(true, true)))
	mux.Handle(authConfig.SessionsPath, authConfig.SessionHandler())
	mux.Handle("GET /settings", srvu.With(SettingsPage(view, db), authConfig.Middleware(false, false)))

	HandleNested(mux, "/invites/", auth.InviteHandler(&InviteStore{db: db, view: view}, authConfig))
	mux.Handle("/chore-lists/", srvu.With(ChoreListMux(db, view), authConfig.Middleware(false, false)))
	mux.Handle("/chores/", srvu.With(ChoreMux(db, view), authConfig.Middleware(false, false)))
	mux.Handle("/{$}", http.RedirectHandler("/chore-lists/", http.StatusFound))
	return mux
}

func Run(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, getEnv func(string) string, getwd func() (string, error)) error {
	cfg, err := parseConfig(args[1:], getEnv)
	if err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, os.Kill)
	defer cancel()
	logger := srvu.LogToOutput(log.New(stdout, "", log.LstdFlags|log.Lshortfile))

	db, err := GetMigratedDB(ctx, choretracker.StaticEmbeddedFS, "static/migrations", cfg.DbURL)
	if err != nil {
		return fmt.Errorf("failed to migrate db: %w", err)
	}

	public, tmplProv, err := templ.GetPublicAndTemplates(choretracker.StaticEmbeddedFS, &templ.Config{
		Watch:        cfg.Watch,
		TmplPatterns: []string{"templates/*.gohtml"},
	})
	view := &View{p: tmplProv}
	if err != nil {
		return fmt.Errorf("sub static: %w", err)
	}
	authConfig := auth.Config{
		Provider:                    &AuthProvider{db: db, view: view},
		RedirectParam:               "redirect",
		UnauthorizedRedirect:        "/login",
		DefaultLogoutRedirect:       "/login",
		LoginFailedRedirect:         "/login",
		DefaultLoginSuccessRedirect: "/",
		SessionsPath:                "/sessions/",
		SessionCookie: auth.CookieConfig{
			Name:          "session",
			Expire:        5 * time.Second,
			RefreshMargin: 2 * time.Second,
			TokenLength:   32,
			Store:         auth.NewInMemoryTokenStore(),
		},
		RefreshCookie: auth.CookieConfig{
			Name:        "refresh_session",
			Expire:      24 * time.Hour * 30,
			TokenLength: 102,
			Store:       &DBTokenStore{DB: db},
		},
	}

	mux := http.NewServeMux()
	HandleNested(mux, "GET /static/public/", srvu.With(http.FileServerFS(public), srvu.WithCacheCtrlHeader(365*24*time.Hour)))
	mux.Handle("/", Mux(db, view, authConfig))

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
