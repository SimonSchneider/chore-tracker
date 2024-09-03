package chore

import (
	"context"
	"database/sql"
	"errors"
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
	"time"
)

type flags struct {
	addr   string
	server bool
	watch  bool
}

func parseFlags(args []string) (flags, error) {
	fSet := flag.NewFlagSet("server", flag.ExitOnError)
	cfg := flags{}
	fSet.StringVar(&cfg.addr, "addr", "", "server address")
	fSet.BoolVar(&cfg.server, "server", false, "run as server")
	fSet.BoolVar(&cfg.watch, "watch", false, "watch for changes")
	if err := fSet.Parse(args[1:]); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func getFS(watch bool) fs.FS {
	if _, err := os.Stat("static"); err != nil {
		panic(err)
	}
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

	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		return fmt.Errorf("opening db: %w", err)
	}
	if err := Setup(ctx, db); err != nil {
		return fmt.Errorf("setup db: %w", err)
	}

	var mux http.Handler
	if cfg.server {
		tmplProv := newTemplateProvider(getFS(cfg.watch), cfg.watch)
		mux = NewHtmlMux(db, tmplProv)
	} else {
		mux = NewMux(db)
	}

	srv := &http.Server{
		BaseContext: func(listener net.Listener) context.Context {
			return ctx
		},
		Addr:    cfg.addr,
		Handler: srvu.WithCompression(mux),
	}
	go func() {
		<-ctx.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		logger.Printf("Shutdown requested, shutting down gracefully")
		if err := srv.Shutdown(ctx); err != nil {
			logger.Printf("Shutdown timed out, killing server forcefully")
			srv.Close()
		}
	}()
	logger.Printf("listening on %s", cfg.addr)
	if err := srv.ListenAndServe(); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
	return nil
}
