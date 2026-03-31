package sync_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/theakshaypant/mission-control/internal/core"
	"github.com/theakshaypant/mission-control/internal/store/jsonfile"
	syncp "github.com/theakshaypant/mission-control/internal/sync"
	"github.com/theakshaypant/mission-control/internal/testutil"
)

func openStore(t *testing.T) core.Store {
	t.Helper()
	s, err := jsonfile.Open(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	return s
}

func makeItem(id string, updatedAt time.Time) core.Item {
	return core.Item{
		ID:        id,
		Source:    "test",
		Type:      "task",
		UpdatedAt: updatedAt,
	}
}

// TestSyncAll_UpsertsItems verifies that items returned by a source are
// written to the store.
func TestSyncAll_UpsertsItems(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()

	now := time.Now()
	src := &testutil.MockSource{
		NameVal: "src",
		Items:   []core.Item{makeItem("test:ns#1", now), makeItem("test:ns#2", now)},
	}

	runner := syncp.New(store, []core.Source{src})
	if err := runner.SyncAll(ctx); err != nil {
		t.Fatalf("SyncAll: %v", err)
	}

	items, err := store.ListItems(ctx, core.ItemFilter{})
	if err != nil {
		t.Fatalf("ListItems: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items in store, got %d", len(items))
	}
}

// TestSyncAll_RecordsCursor verifies that the sync cursor is stored after
// a successful sync and passed to the next Sync call.
func TestSyncAll_RecordsCursor(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()

	src := &testutil.MockSource{NameVal: "src"}
	runner := syncp.New(store, []core.Source{src})

	// First sync — no cursor yet.
	if err := runner.SyncAll(ctx); err != nil {
		t.Fatalf("first SyncAll: %v", err)
	}
	if src.LastSince != nil {
		t.Errorf("first sync: expected nil since, got %v", src.LastSince)
	}

	// Second sync — cursor from first sync should be passed in.
	if err := runner.SyncAll(ctx); err != nil {
		t.Fatalf("second SyncAll: %v", err)
	}
	if src.LastSince == nil {
		t.Error("second sync: expected non-nil since from stored cursor")
	}
}

// TestSyncAll_AdvancesLastInteractedAt verifies that UserActivityAt on a
// returned item is written into ItemState.LastInteractedAt.
func TestSyncAll_AdvancesLastInteractedAt(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()

	activityAt := time.Now().Add(-time.Hour)
	item := makeItem("test:ns#1", time.Now())
	item.UserActivityAt = &activityAt

	src := &testutil.MockSource{NameVal: "src", Items: []core.Item{item}}
	runner := syncp.New(store, []core.Source{src})
	if err := runner.SyncAll(ctx); err != nil {
		t.Fatalf("SyncAll: %v", err)
	}

	state, err := store.GetItemState(ctx, item.ID)
	if err != nil {
		t.Fatalf("GetItemState: %v", err)
	}
	if state == nil || state.LastInteractedAt == nil {
		t.Fatal("expected LastInteractedAt to be set")
	}
	if !state.LastInteractedAt.Equal(activityAt) {
		t.Errorf("LastInteractedAt: got %v, want %v", state.LastInteractedAt, activityAt)
	}
}

// TestSyncAll_DoesNotRegressLastInteractedAt verifies that an older
// UserActivityAt does not overwrite a more recent stored LastInteractedAt.
func TestSyncAll_DoesNotRegressLastInteractedAt(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()

	recent := time.Now()
	older := recent.Add(-2 * time.Hour)

	// Pre-seed a recent state.
	store.SetItemState(ctx, core.ItemState{ItemID: "test:ns#1", LastInteractedAt: &recent})

	// Source reports older activity.
	item := makeItem("test:ns#1", time.Now())
	item.UserActivityAt = &older

	src := &testutil.MockSource{NameVal: "src", Items: []core.Item{item}}
	runner := syncp.New(store, []core.Source{src})
	if err := runner.SyncAll(ctx); err != nil {
		t.Fatalf("SyncAll: %v", err)
	}

	state, _ := store.GetItemState(ctx, "test:ns#1")
	if !state.LastInteractedAt.Equal(recent) {
		t.Errorf("expected LastInteractedAt to remain %v, got %v", recent, state.LastInteractedAt)
	}
}

// TestSyncAll_SourceError verifies that a source error is propagated and
// wrapped with the source name.
func TestSyncAll_SourceError(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()

	syncErr := errors.New("rate limited")
	src := &testutil.MockSource{NameVal: "bad-src", SyncErr: syncErr}
	runner := syncp.New(store, []core.Source{src})

	err := runner.SyncAll(ctx)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, syncErr) {
		t.Errorf("expected wrapped syncErr, got: %v", err)
	}
}
