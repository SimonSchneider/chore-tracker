package httpu

import (
	"net/http"
	"net/url"
	"strings"
)

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

func GetReferer(r *http.Request, fb string) string {
	ref, err := url.Parse(r.Referer())
	if err != nil {
		return ""
	}
	return ref.RequestURI()
}

func RedirectToReferer(w http.ResponseWriter, r *http.Request, fb string) {
	http.Redirect(w, r, GetReferer(r, fb), http.StatusSeeOther)
}

func RedirectToNext(w http.ResponseWriter, r *http.Request, fb string) {
	next := r.FormValue("next")
	if next == "" {
		next = fb
	}
	http.Redirect(w, r, next, http.StatusSeeOther)
}
