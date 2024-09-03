package srvu

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func Encode(w http.ResponseWriter, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		return Err(http.StatusInternalServerError, fmt.Errorf("failed to encode: %w", err))
	}
	return nil
}

type FormParser interface {
	FromForm(r *http.Request) error
}

func Decode(r *http.Request, t any, acceptEmpty bool) error {
	ct := r.Header.Get("Content-Type")
	fmt.Printf("Content-Type: %s, acceptEmpty: %s\n", ct, acceptEmpty)
	switch ct {
	case "application/json":
		if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
			return Err(http.StatusUnsupportedMediaType, fmt.Errorf("failed to decode: %w", err))
		}
		return nil
	case "multipart/form-data", "application/x-www-form-urlencoded":
		if p, ok := t.(FormParser); ok {
			if err := p.FromForm(r); err != nil {
				return Err(http.StatusBadRequest, fmt.Errorf("failed to parse form: %w", err))
			}
		}
		return Err(http.StatusUnsupportedMediaType, fmt.Errorf("unsupported content type: %s", ct))
	case "":
		if acceptEmpty {
			return nil
		}
	}
	return Err(http.StatusUnsupportedMediaType, fmt.Errorf("unsupported content type: %s", ct))
}
