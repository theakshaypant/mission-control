package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/theakshaypant/mission-control/internal/actions"
	"github.com/theakshaypant/mission-control/internal/core"
)

type itemsHandler struct {
	actions *actions.Actions
}

func newItemsHandler(a *actions.Actions) *itemsHandler {
	return &itemsHandler{actions: a}
}

// list handles GET /items with optional query params:
// ?needs_attention=true, ?waits_on_me=true, ?source=<kind>, ?type=<type> (repeatable).
func (h *itemsHandler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := core.ItemFilter{
		NeedsAttention: q.Get("needs_attention") == "true",
		WaitsOnMe:      q.Get("waits_on_me") == "true",
		Source:         core.SourceKind(q.Get("source")),
		SourceName:     q.Get("source_name"),
	}
	for _, t := range q["type"] {
		filter.Types = append(filter.Types, core.ItemType(t))
	}
	items, err := h.actions.ListItems(r.Context(), filter)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// summary handles GET /summary — shorthand for items with NeedsAttention=true.
func (h *itemsHandler) summary(w http.ResponseWriter, r *http.Request) {
	items, err := h.actions.Summary(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// dismiss handles POST /items/{id}/dismiss.
func (h *itemsHandler) dismiss(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.actions.DismissItem(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// snoozeRequest is the JSON body for POST /items/{id}/snooze.
// Exactly one of For or Until must be present.
type snoozeRequest struct {
	For   string `json:"for"`   // duration string, e.g. "24h" or "7d"
	Until string `json:"until"` // RFC3339 timestamp
}

// snooze handles POST /items/{id}/snooze.
func (h *itemsHandler) snooze(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req snoozeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorBody{Error: "invalid request body"})
		return
	}

	until, err := parseSnoozeRequest(req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorBody{Error: err.Error()})
		return
	}

	if err := h.actions.SnoozeItem(r.Context(), id, until); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// parseSnoozeRequest converts a snoozeRequest into an absolute time.
// "for" accepts Go durations and days (e.g. "2h30m", "24h", "7d").
// "until" accepts RFC3339, date (e.g. "2026-04-01"), or HH:MM (today at that time, local).
func parseSnoozeRequest(req snoozeRequest) (time.Time, error) {
	hasFor := req.For != ""
	hasUntil := req.Until != ""
	switch {
	case hasFor && hasUntil:
		return time.Time{}, fmt.Errorf("provide either 'for' or 'until', not both")
	case !hasFor && !hasUntil:
		return time.Time{}, fmt.Errorf("one of 'for' or 'until' is required")
	case hasFor:
		d, err := parseAPIDuration(req.For)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid 'for' value %q: %w", req.For, err)
		}
		return time.Now().Add(d), nil
	default: // hasUntil
		return parseAPIUntil(req.Until)
	}
}

// parseAPIDuration extends time.ParseDuration to support "d" (days).
func parseAPIDuration(s string) (time.Duration, error) {
	if strings.HasSuffix(s, "d") {
		var days int
		if _, err := fmt.Sscanf(strings.TrimSuffix(s, "d"), "%d", &days); err != nil {
			return 0, fmt.Errorf("invalid day count in %q", s)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}

// parseAPIUntil parses an until string in RFC3339, date-only, or HH:MM format.
func parseAPIUntil(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.DateOnly, s); err == nil {
		now := time.Now()
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, now.Location()), nil
	}
	if t, err := time.Parse("15:04", s); err == nil {
		now := time.Now()
		return time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, now.Location()), nil
	}
	return time.Time{}, fmt.Errorf(
		"invalid 'until' value %q: accepted formats are HH:MM (e.g. 14:30), date (e.g. 2026-04-01), or RFC3339",
		s,
	)
}
