package chore

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"github.com/SimonSchneider/go-testing/srvu"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
)

type flags struct {
	addr  string
	watch bool
	db    string
}

func parseFlags(args []string) (flags, error) {
	fSet := flag.NewFlagSet("server", flag.ExitOnError)
	cfg := flags{}
	fSet.StringVar(&cfg.addr, "addr", "", "server address")
	fSet.BoolVar(&cfg.watch, "watch", false, "watch for changes")
	fSet.StringVar(&cfg.db, "db", "file::memory:?cache=shared", "db connection string")
	if err := fSet.Parse(args[1:]); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func getFS(watch bool) fs.FS {
	if watch {
		if _, err := os.Stat("static"); err == nil {
			return os.DirFS("static")
		}
	}
	// TODO: grab from embeddedFS
	return os.DirFS("static")
}

func Run(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, getEnv func(string) string, getwd func() (string, error)) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, os.Kill)
	defer cancel()
	cfg, err := parseFlags(args)
	if err != nil {
		return err
	}
	logger := log.New(stdout, "", log.LstdFlags)

	db, err := sql.Open("sqlite3", cfg.db)
	if err != nil {
		return fmt.Errorf("opening db: %w", err)
	}
	if err := Setup(ctx, db); err != nil {
		return fmt.Errorf("setup db: %w", err)
	}

	var mux http.Handler
	files := getFS(cfg.watch)
	tmplProv := srvu.NewTemplateProvider(files, cfg.watch)
	staticFiles, err := fs.Sub(files, "public")
	if err != nil {
		return fmt.Errorf("sub static: %w", err)
	}
	mux = NewHtmlMux(db, staticFiles, tmplProv)

	srv := &http.Server{
		BaseContext: func(listener net.Listener) context.Context {
			return ctx
		},
		Addr:    cfg.addr,
		Handler: srvu.With(mux, srvu.WithCompression(), srvu.WithLogger(logger)),
	}
	logger.Printf("starting chore server, listening on %s\n  sqliteDB: %s", cfg.addr, cfg.db)
	return srvu.RunServerGracefully(ctx, srv, logger)
}
