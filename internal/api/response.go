package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/theakshaypant/mission-control/internal/actions"
)

// ErrorBody is the JSON shape for all error responses.
type ErrorBody struct {
	Error string `json:"error"`
}

// writeJSON encodes v as JSON with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError maps an error to an HTTP status code and writes a JSON error body.
// actions.ErrNotFound → 404; all others → 500.
func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	if errors.Is(err, actions.ErrNotFound) {
		status = http.StatusNotFound
	}
	writeJSON(w, status, ErrorBody{Error: err.Error()})
}
