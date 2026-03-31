package jsonfile_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/theakshaypant/mission-control/internal/core"
	"github.com/theakshaypant/mission-control/internal/store/jsonfile"
)

// Compile-time check that *Store satisfies core.Store.
var _ core.Store = (*jsonfile.Store)(nil)

func openStore(t *testing.T) core.Store {
	t.Helper()
	s, err := jsonfile.Open(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return s
}

func makeItem(id string, updatedAt time.Time) core.Item {
	return core.Item{
		ID:        id,
		Source:    "github",
		Type:      "pr",
		Namespace: "owner/repo",
		UpdatedAt: updatedAt,
		WaitsOnMe: true,
	}
}

// TestOpenCreatesFile verifies that opening a non-existent path works and
// that the file is created on the first mutation.
func TestOpenCreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "state.json")
	s, err := jsonfile.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	item := makeItem("github:owner/repo#1", time.Now())
	if err := s.UpsertItem(context.Background(), item); err != nil {
		t.Fatalf("UpsertItem: %v", err)
	}

	// Reload from disk and verify the item survived.
	s2, err := jsonfile.Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	items, err := s2.ListItems(context.Background(), core.ItemFilter{})
	if err != nil {
		t.Fatalf("ListItems: %v", err)
	}
	if len(items) != 1 || items[0].ID != item.ID {
		t.Errorf("expected item %q to survive reload, got %v", item.ID, items)
	}
}

// TestUpsertItemRoundtrip verifies that UpsertItem persists all fields and
// that a subsequent open reads them back correctly.
func TestUpsertItemRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	now := time.Now().Truncate(time.Second)
	item := core.Item{
		ID:        "github:owner/repo#42",
		Source:    "github",
		Type:      "pr",
		Title:     "Fix the thing",
		URL:       "https://github.com/owner/repo/pull/42",
		Namespace: "owner/repo",
		WaitsOnMe: true,
		IsAssigned: true,
		CreatedAt: now.Add(-time.Hour),
		UpdatedAt: now,
	}

	s, _ := jsonfile.Open(path)
	if err := s.UpsertItem(context.Background(), item); err != nil {
		t.Fatalf("UpsertItem: %v", err)
	}

	s2, _ := jsonfile.Open(path)
	items, err := s2.ListItems(context.Background(), core.ItemFilter{})
	if err != nil {
		t.Fatalf("ListItems: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	got := items[0]
	if got.ID != item.ID {
		t.Errorf("ID: got %q, want %q", got.ID, item.ID)
	}
	if got.Title != item.Title {
		t.Errorf("Title: got %q, want %q", got.Title, item.Title)
	}
	if !got.WaitsOnMe {
		t.Error("WaitsOnMe: expected true")
	}
	if !got.UpdatedAt.Equal(item.UpdatedAt) {
		t.Errorf("UpdatedAt: got %v, want %v", got.UpdatedAt, item.UpdatedAt)
	}
}

// TestUpsertItemOverwrites verifies that upserting with the same ID replaces
// the existing record.
func TestUpsertItemOverwrites(t *testing.T) {
	s := openStore(t)
	ctx := context.Background()

	now := time.Now()
	orig := makeItem("github:owner/repo#1", now)
	updated := orig
	updated.Title = "updated title"
	updated.UpdatedAt = now.Add(time.Minute)

	if err := s.UpsertItem(ctx, orig); err != nil {
		t.Fatalf("first UpsertItem: %v", err)
	}
	if err := s.UpsertItem(ctx, updated); err != nil {
		t.Fatalf("second UpsertItem: %v", err)
	}

	items, _ := s.ListItems(ctx, core.ItemFilter{})
	if len(items) != 1 {
		t.Fatalf("expected 1 item after upsert, got %d", len(items))
	}
	if items[0].Title != "updated title" {
		t.Errorf("expected updated title, got %q", items[0].Title)
	}
}

// TestListItemsFilterBySource verifies that Source filtering works.
func TestListItemsFilterBySource(t *testing.T) {
	s := openStore(t)
	ctx := context.Background()

	ghItem := makeItem("github:owner/repo#1", time.Now())
	ghItem.Source = "github"
	jiraItem := makeItem("jira:PROJ-1", time.Now())
	jiraItem.Source = "jira"

	s.UpsertItem(ctx, ghItem)
	s.UpsertItem(ctx, jiraItem)

	items, err := s.ListItems(ctx, core.ItemFilter{Source: "github"})
	if err != nil {
		t.Fatalf("ListItems: %v", err)
	}
	if len(items) != 1 || items[0].ID != ghItem.ID {
		t.Errorf("expected only github item, got %v", items)
	}
}

// TestListItemsFilterByType verifies that Types filtering works.
func TestListItemsFilterByType(t *testing.T) {
	s := openStore(t)
	ctx := context.Background()

	pr := makeItem("github:owner/repo#1", time.Now())
	pr.Type = "pr"
	issue := makeItem("github:owner/repo#2", time.Now())
	issue.Type = "issue"

	s.UpsertItem(ctx, pr)
	s.UpsertItem(ctx, issue)

	items, err := s.ListItems(ctx, core.ItemFilter{Types: []core.ItemType{"issue"}})
	if err != nil {
		t.Fatalf("ListItems: %v", err)
	}
	if len(items) != 1 || items[0].ID != issue.ID {
		t.Errorf("expected only issue item, got %v", items)
	}
}

// TestListItemsFilterByWaitsOnMe verifies that WaitsOnMe filtering works.
func TestListItemsFilterByWaitsOnMe(t *testing.T) {
	s := openStore(t)
	ctx := context.Background()

	actionable := makeItem("github:owner/repo#1", time.Now())
	actionable.WaitsOnMe = true
	passive := makeItem("github:owner/repo#2", time.Now())
	passive.WaitsOnMe = false

	s.UpsertItem(ctx, actionable)
	s.UpsertItem(ctx, passive)

	items, err := s.ListItems(ctx, core.ItemFilter{WaitsOnMe: true})
	if err != nil {
		t.Fatalf("ListItems: %v", err)
	}
	if len(items) != 1 || items[0].ID != actionable.ID {
		t.Errorf("expected only actionable item, got %v", items)
	}
}

// TestListItemsNeedsAttention verifies that NeedsAttention filtering
// excludes dismissed and snoozed items, and includes new or updated ones.
func TestListItemsNeedsAttention(t *testing.T) {
	s := openStore(t)
	ctx := context.Background()

	now := time.Now()

	// New item — no state, needs attention.
	newItem := makeItem("github:owner/repo#1", now)

	// Dismissed item — should not appear.
	dismissed := makeItem("github:owner/repo#2", now)

	// Snoozed item — should not appear.
	snoozed := makeItem("github:owner/repo#3", now)

	// Source says nothing waits on user — should not appear.
	noSignal := makeItem("github:owner/repo#4", now)
	noSignal.WaitsOnMe = false

	// WaitsOnMe with no state override — should appear.
	pending := makeItem("github:owner/repo#5", now)

	for _, item := range []core.Item{newItem, dismissed, snoozed, noSignal, pending} {
		s.UpsertItem(ctx, item)
	}

	futureSnooze := now.Add(time.Hour)

	s.SetItemState(ctx, core.ItemState{ItemID: dismissed.ID, Dismissed: true})
	s.SetItemState(ctx, core.ItemState{ItemID: snoozed.ID, SnoozedUntil: &futureSnooze})

	items, err := s.ListItems(ctx, core.ItemFilter{NeedsAttention: true})
	if err != nil {
		t.Fatalf("ListItems: %v", err)
	}

	wantIDs := map[string]bool{newItem.ID: true, pending.ID: true}
	if len(items) != len(wantIDs) {
		t.Errorf("expected %d items, got %d: %v", len(wantIDs), len(items), itemIDs(items))
	}
	for _, item := range items {
		if !wantIDs[item.ID] {
			t.Errorf("unexpected item in NeedsAttention result: %s", item.ID)
		}
	}
}

// TestSyncCursorRoundtrip verifies that sync cursors survive a store reload.
func TestSyncCursorRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)
	s, _ := jsonfile.Open(path)
	if err := s.SetLastSyncedAt(ctx, "work-github", now); err != nil {
		t.Fatalf("SetLastSyncedAt: %v", err)
	}

	s2, _ := jsonfile.Open(path)
	got, err := s2.GetLastSyncedAt(ctx, "work-github")
	if err != nil {
		t.Fatalf("GetLastSyncedAt: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil time after reload")
	}
	if !got.Equal(now) {
		t.Errorf("sync cursor: got %v, want %v", got, now)
	}
}

// TestGetLastSyncedAtNeverSynced verifies that an unknown source returns nil, nil.
func TestGetLastSyncedAtNeverSynced(t *testing.T) {
	s := openStore(t)
	got, err := s.GetLastSyncedAt(context.Background(), "unknown-source")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for unknown source, got %v", got)
	}
}

// TestItemStateRoundtrip verifies that SetItemState and GetItemState persist
// across store reloads.
func TestItemStateRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)
	future := now.Add(time.Hour)
	state := core.ItemState{
		ItemID:           "github:owner/repo#7",
		LastInteractedAt: &now,
		Dismissed:        false,
		SnoozedUntil:     &future,
	}

	s, _ := jsonfile.Open(path)
	if err := s.SetItemState(ctx, state); err != nil {
		t.Fatalf("SetItemState: %v", err)
	}

	s2, _ := jsonfile.Open(path)
	got, err := s2.GetItemState(ctx, state.ItemID)
	if err != nil {
		t.Fatalf("GetItemState: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil state after reload")
	}
	if got.ItemID != state.ItemID {
		t.Errorf("ItemID: got %q, want %q", got.ItemID, state.ItemID)
	}
	if !got.LastInteractedAt.Equal(*state.LastInteractedAt) {
		t.Errorf("LastInteractedAt: got %v, want %v", got.LastInteractedAt, state.LastInteractedAt)
	}
	if !got.SnoozedUntil.Equal(*state.SnoozedUntil) {
		t.Errorf("SnoozedUntil: got %v, want %v", got.SnoozedUntil, state.SnoozedUntil)
	}
}

// TestGetItemStateUnknown verifies that an unknown item returns nil, nil.
func TestGetItemStateUnknown(t *testing.T) {
	s := openStore(t)
	got, err := s.GetItemState(context.Background(), "github:owner/repo#999")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for unknown item, got %v", got)
	}
}

// TestSetItemStateOverwrites verifies that setting state twice replaces the first.
func TestSetItemStateOverwrites(t *testing.T) {
	s := openStore(t)
	ctx := context.Background()
	id := "github:owner/repo#10"

	s.SetItemState(ctx, core.ItemState{ItemID: id, Dismissed: true})
	s.SetItemState(ctx, core.ItemState{ItemID: id, Dismissed: false})

	got, _ := s.GetItemState(ctx, id)
	if got == nil {
		t.Fatal("expected non-nil state")
	}
	if got.Dismissed {
		t.Error("expected Dismissed=false after overwrite")
	}
}

// TestNeedsAttentionLogic exercises the NeedsAttention method directly
// to document its contract independently of the store.
func TestNeedsAttentionLogic(t *testing.T) {
	now := time.Now()
	item := makeItem("github:owner/repo#1", now)

	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	cases := []struct {
		name  string
		item  core.Item
		state *core.ItemState
		want  bool
	}{
		{"nil state → needs attention", item, nil, true},
		{"no prior interaction", item, &core.ItemState{}, true},
		{"dismissed → no attention", item, &core.ItemState{Dismissed: true}, false},
		{"snoozed in future → no attention", item, &core.ItemState{SnoozedUntil: &future}, false},
		{"snooze expired → needs attention", item, &core.ItemState{SnoozedUntil: &past}, true},
		{"WaitsOnMe false → no attention", func() core.Item { i := item; i.WaitsOnMe = false; return i }(), nil, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.item.NeedsAttention(tc.state); got != tc.want {
				t.Errorf("NeedsAttention = %v, want %v", got, tc.want)
			}
		})
	}
}

func itemIDs(items []core.Item) []string {
	ids := make([]string, len(items))
	for i, item := range items {
		ids[i] = item.ID
	}
	return ids
}
