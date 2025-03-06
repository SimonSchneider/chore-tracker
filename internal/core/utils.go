package core

import (
	"github.com/SimonSchneider/goslu/sid"
	"net/http"
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
