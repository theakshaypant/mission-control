package api

import (
	"net/http"

	"github.com/theakshaypant/mission-control/internal/actions"
)

type syncHandler struct {
	actions *actions.Actions
}

func newSyncHandler(a *actions.Actions) *syncHandler {
	return &syncHandler{actions: a}
}

// syncAll handles POST /sync — triggers a full sync of all sources.
func (h *syncHandler) syncAll(w http.ResponseWriter, r *http.Request) {
	if err := h.actions.SyncAll(r.Context()); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// syncSource handles POST /sync/{source} — triggers a sync for a named source.
func (h *syncHandler) syncSource(w http.ResponseWriter, r *http.Request) {
	source := r.PathValue("source")
	if err := h.actions.SyncSource(r.Context(), source); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// status handles GET /sync/status — returns last sync time per source.
func (h *syncHandler) status(w http.ResponseWriter, r *http.Request) {
	statuses, err := h.actions.SyncStatus(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, statuses)
}
