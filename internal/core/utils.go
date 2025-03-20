package core

import (
	"github.com/SimonSchneider/goslu/sid"
	"net/http"
	"net/url"
	"strings"
)

func NewId() string {
	return sid.MustNewString(15)
}

func HandleNested(mux *http.ServeMux, pattern string, h http.Handler) {
	pathStart := strings.Index(pattern, "/")
	if pathStart == -1 {
		mux.Handle(pattern, h)
		return
	}
	prefix := pattern[pathStart:]
	if len(prefix) == 0 || prefix == "/" {
		mux.Handle(pattern, h)
		return
	}
	if prefix[len(prefix)-1] == '/' {
		prefix = prefix[:len(prefix)-1]
	}
	mux.Handle(pattern, http.StripPrefix(prefix, h))
}

func getReferer(r *http.Request, fb string) string {
	ref, err := url.Parse(r.Referer())
	if err != nil {
		return ""
	}
	return ref.RequestURI()
}

func redirectToReferer(w http.ResponseWriter, r *http.Request, fb string) {
	http.Redirect(w, r, getReferer(r, fb), http.StatusSeeOther)
}

func redirectToNext(w http.ResponseWriter, r *http.Request, fb string) {
	next := r.URL.Query().Get("next")
	if next == "" {
		next = fb
	}
	http.Redirect(w, r, next, http.StatusSeeOther)
}
