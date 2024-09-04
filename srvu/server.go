package srvu

import (
	"context"
	"errors"
	"net/http"
	"time"
)

type Middleware func(http.Handler) http.Handler

func RunServerGracefully(ctx context.Context, srv *http.Server, logger Logger) error {
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
	if err := srv.ListenAndServe(); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
	return nil
}

func With(h http.Handler, mws ...Middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}
