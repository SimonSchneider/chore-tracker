package srvu

import (
	"context"
	"fmt"
	"net/http"
)

type capturingResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *capturingResponseWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

func (w *capturingResponseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func WithLogger(logger Logger) Middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodHead {
				logger.Printf("request: %s %s", r.Method, r.URL)
			}
			r = r.WithContext(ContextWithLogger(r.Context(), logger))
			cw := &capturingResponseWriter{ResponseWriter: w}
			h.ServeHTTP(cw, r)
			//if r.Method != http.MethodHead {
			logger.Printf("response: %s %s %d", r.Method, r.URL, cw.status)
			//}
		})
	}
}

type Outputer interface {
	Output(calldepth int, s string) error
}

func LogToOutput(std Outputer) Logger {
	return LoggerFunc(func(s string, a ...any) {
		std.Output(3, fmt.Sprintf(s, a...))
	})
}

type Logger interface {
	Printf(string, ...any)
}

type LoggerFunc func(string, ...any)

func (f LoggerFunc) Printf(format string, args ...any) {

	f(format, args...)
}

func NewLoggerFunc(f func(string, ...any)) Logger {
	return LoggerFunc(f)
}

type nilLogger struct{}

func (nilLogger) Printf(string, ...any) {}

const loggerKey = "x-logger"

func ContextWithLogger(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

func GetLogger(ctx context.Context) Logger {
	if l, ok := ctx.Value(loggerKey).(Logger); ok {
		return l
	}
	return nilLogger{}
}
