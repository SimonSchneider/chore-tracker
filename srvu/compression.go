package srvu

import (
	"compress/gzip"
	"net/http"
	"strings"
)

type GzipResponseWriter struct {
	http.ResponseWriter
	gz *gzip.Writer
}

func NewGzipResponseWriter(w http.ResponseWriter) *GzipResponseWriter {
	return &GzipResponseWriter{ResponseWriter: w, gz: gzip.NewWriter(w)}
}

func (w *GzipResponseWriter) Write(b []byte) (int, error) {
	return w.gz.Write(b)
}

func (w *GzipResponseWriter) Close() error {
	return w.gz.Close()
}

func WithCompression(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			w.Header().Set("Content-Encoding", "gzip")
			gz := NewGzipResponseWriter(w)
			defer gz.Close()
			h.ServeHTTP(gz, r)
		}
		h.ServeHTTP(w, r)
	})
}
