package bla

import (
	"encoding/json"
	"net/http"
	"strings"
)

type Req struct {
	Name string `json:"name"`
}

func (r *Req) FromForm(req *http.Request) error {
	r.Name = req.FormValue("name")
	return nil
}

func HelloHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req Req
		if err := Decode(r, &req); err != nil {
			http.Error(w, err.Error(), http.StatusUnsupportedMediaType)
			return
		}
		if req.Name == "" {
			req.Name = "Simon"
		}
		accept := r.Header.Get("Accept")
		if strings.Contains(accept, "application/json") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"message": "hello " + req.Name})
		} else if strings.Contains(accept, "text/html") {
			w.Header().Set("Content-Type", "text/html")
		} else {
			http.Error(w, "unsupported accept header", http.StatusNotAcceptable)
		}
	}
}
