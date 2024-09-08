package srvu

import (
	"context"
	"errors"
	"net/http"
)

type HandlerFunc func(context.Context, http.ResponseWriter, *http.Request) error

type Handler interface {
	ServeHTTP(context.Context, http.ResponseWriter, *http.Request) error
}

func (h HandlerFunc) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	return h(ctx, w, r)
}

func ErrHandlerFunc(h HandlerFunc) http.Handler {
	return ErrHandler(h)
}

func ErrHandler(h Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := h.ServeHTTP(r.Context(), w, r); err != nil {
			var serr StatusError
			if errors.As(err, &serr) {
				logger := GetLogger(r.Context())
				logger.Printf("error: %s", serr.Error())
				http.Error(w, serr.Error(), serr.Code)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
}

type StatusError struct {
	Code int
	Err  error
}

func Err(code int, err error) error {
	return StatusError{Code: code, Err: err}
}

func ErrStr(code int, err string) error {
	return StatusError{Code: code, Err: errors.New(err)}
}

func (s StatusError) Error() string {
	return s.Err.Error()
}

func (s StatusError) Unwrap() error {
	return s.Err
}
