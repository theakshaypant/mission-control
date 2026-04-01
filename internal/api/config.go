package api

import (
	"context"
	"io"
	"net/http"
)

type configHandler struct {
	getYAML func() (string, error)
	reload  func(ctx context.Context, yaml string) error
}

func newConfigHandler(
	getYAML func() (string, error),
	reload func(ctx context.Context, yaml string) error,
) *configHandler {
	return &configHandler{getYAML: getYAML, reload: reload}
}

func (h *configHandler) get(w http.ResponseWriter, r *http.Request) {
	yaml, err := h.getYAML()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorBody{Error: err.Error()})
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(yaml))
}

func (h *configHandler) put(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorBody{Error: "could not read request body"})
		return
	}
	if err := h.reload(r.Context(), string(body)); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorBody{Error: err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
