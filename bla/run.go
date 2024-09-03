package bla

import (
	"context"
	"errors"
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"
)

type flags struct {
	addr string
}

func parseFlags(args []string) (flags, error) {
	fSet := flag.NewFlagSet("bla", flag.ExitOnError)
	cfg := flags{}
	fSet.StringVar(&cfg.addr, "addr", "", "server address")
	if err := fSet.Parse(args[1:]); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func Run(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, getEnv func(string) string, getwd func() (string, error)) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, os.Kill)
	defer cancel()
	cfg, err := parseFlags(args)
	if err != nil {
		return err
	}
	logger := log.New(stdout, "", log.LstdFlags)

	mux := http.NewServeMux()
	mux.Handle("GET /", HelloHandler())
	srv := &http.Server{
		BaseContext: func(listener net.Listener) context.Context {
			return ctx
		},
		Addr:    cfg.addr,
		Handler: mux,
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
