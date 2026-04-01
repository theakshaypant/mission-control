package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/theakshaypant/mission-control/internal/actions"
	"github.com/theakshaypant/mission-control/internal/api"
	"github.com/theakshaypant/mission-control/internal/core"
	"github.com/theakshaypant/mission-control/internal/store/jsonfile"
	syncp "github.com/theakshaypant/mission-control/internal/sync"
	"github.com/theakshaypant/mission-control/internal/testutil"
)

func newTestServer(t *testing.T, items ...core.Item) (*api.Server, core.Store) {
	t.Helper()
	store, err := jsonfile.Open(filepath.Join(t.TempDir(), "state.json"))
	require.NoError(t, err)

	ctx := context.Background()
	for _, item := range items {
		require.NoError(t, store.UpsertItem(ctx, item))
	}
	src := &testutil.MockSource{NameVal: "test-src"}
	runner := syncp.New(store, []core.Source{src})
	a := actions.New(store, runner)
	return api.New(":0", a, nil), store
}

func makeItem(id string) core.Item {
	return core.Item{
		ID:        id,
		Source:    "github",
		Type:      "pr",
		Namespace: "owner/repo",
		UpdatedAt: time.Now(),
		WaitsOnMe: true,
	}
}

func do(t *testing.T, srv *api.Server, method, path string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	return w
}

// --- GET /items ---

func TestGetItems_ReturnsAll(t *testing.T) {
	srv, _ := newTestServer(t, makeItem("github:owner/repo#1"), makeItem("github:owner/repo#2"))
	w := do(t, srv, http.MethodGet, "/items", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	var items []actions.ItemSummary
	require.NoError(t, json.NewDecoder(w.Body).Decode(&items))
	assert.Len(t, items, 2)
}

func TestGetItems_NeedsAttentionFilter(t *testing.T) {
	item := makeItem("github:owner/repo#1")
	srv, store := newTestServer(t, item)

	store.SetItemState(context.Background(), core.ItemState{ItemID: item.ID, Dismissed: true})

	w := do(t, srv, http.MethodGet, "/items?needs_attention=true", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var items []actions.ItemSummary
	require.NoError(t, json.NewDecoder(w.Body).Decode(&items))
	assert.Empty(t, items)
}

// --- GET /summary ---

func TestGetSummary_ReturnsOnlyNeedsAttention(t *testing.T) {
	active := makeItem("github:owner/repo#1")
	dismissed := makeItem("github:owner/repo#2")
	srv, store := newTestServer(t, active, dismissed)

	store.SetItemState(context.Background(), core.ItemState{ItemID: dismissed.ID, Dismissed: true})

	w := do(t, srv, http.MethodGet, "/summary", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var items []actions.ItemSummary
	require.NoError(t, json.NewDecoder(w.Body).Decode(&items))
	require.Len(t, items, 1)
	assert.Equal(t, active.ID, items[0].ID)
}

// --- POST /items/{id}/dismiss ---

func TestDismiss_Success(t *testing.T) {
	item := makeItem("github:owner/repo#1")
	srv, store := newTestServer(t, item)

	w := do(t, srv, http.MethodPost, "/items/github:owner%2Frepo%231/dismiss", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	state, err := store.GetItemState(context.Background(), item.ID)
	require.NoError(t, err)
	require.NotNil(t, state)
	assert.True(t, state.Dismissed)
}

func TestDismiss_NotFound(t *testing.T) {
	srv, _ := newTestServer(t)
	w := do(t, srv, http.MethodPost, "/items/github:owner%2Frepo%23999/dismiss", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- POST /items/{id}/snooze ---

func TestSnooze_WithFor(t *testing.T) {
	item := makeItem("github:owner/repo#1")
	srv, store := newTestServer(t, item)

	body, _ := json.Marshal(map[string]string{"for": "24h"})
	w := do(t, srv, http.MethodPost, "/items/github:owner%2Frepo%231/snooze", body)
	assert.Equal(t, http.StatusOK, w.Code)

	state, err := store.GetItemState(context.Background(), item.ID)
	require.NoError(t, err)
	require.NotNil(t, state)
	require.NotNil(t, state.SnoozedUntil)
	assert.Greater(t, time.Until(*state.SnoozedUntil), 23*time.Hour)
}

func TestSnooze_WithUntilHHMM(t *testing.T) {
	item := makeItem("github:owner/repo#1")
	srv, store := newTestServer(t, item)

	untilStr := time.Now().Add(2 * time.Hour).Format("15:04")
	body, _ := json.Marshal(map[string]string{"until": untilStr})
	w := do(t, srv, http.MethodPost, "/items/github:owner%2Frepo%231/snooze", body)
	assert.Equal(t, http.StatusOK, w.Code)

	state, err := store.GetItemState(context.Background(), item.ID)
	require.NoError(t, err)
	require.NotNil(t, state)
	assert.NotNil(t, state.SnoozedUntil)
}

func TestSnooze_BadRequest_NeitherForNorUntil(t *testing.T) {
	item := makeItem("github:owner/repo#1")
	srv, _ := newTestServer(t, item)

	body, _ := json.Marshal(map[string]string{})
	w := do(t, srv, http.MethodPost, "/items/github:owner%2Frepo%231/snooze", body)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSnooze_BadRequest_BothForAndUntil(t *testing.T) {
	item := makeItem("github:owner/repo#1")
	srv, _ := newTestServer(t, item)

	body, _ := json.Marshal(map[string]string{"for": "24h", "until": "2026-04-01"})
	w := do(t, srv, http.MethodPost, "/items/github:owner%2Frepo%231/snooze", body)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSnooze_NotFound(t *testing.T) {
	srv, _ := newTestServer(t)
	body, _ := json.Marshal(map[string]string{"for": "24h"})
	w := do(t, srv, http.MethodPost, "/items/github:owner%2Frepo%23999/snooze", body)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- POST /sync ---

func TestPostSync_Success(t *testing.T) {
	srv, _ := newTestServer(t)
	w := do(t, srv, http.MethodPost, "/sync", nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPostSyncSource_UnknownSource_Returns500(t *testing.T) {
	srv, _ := newTestServer(t)
	w := do(t, srv, http.MethodPost, "/sync/does-not-exist", nil)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPostSyncSource_KnownSource_Success(t *testing.T) {
	srv, _ := newTestServer(t)
	w := do(t, srv, http.MethodPost, "/sync/test-src", nil)
	assert.Equal(t, http.StatusOK, w.Code)
}
