package actions_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/theakshaypant/mission-control/internal/actions"
	"github.com/theakshaypant/mission-control/internal/core"
	"github.com/theakshaypant/mission-control/internal/store/jsonfile"
	syncp "github.com/theakshaypant/mission-control/internal/sync"
	"github.com/theakshaypant/mission-control/internal/testutil"
)

// newActions builds an Actions backed by a real jsonfile store pre-seeded
// with items, and a MockSource named "test-src".
func newActions(t *testing.T, items ...core.Item) (*actions.Actions, core.Store) {
	t.Helper()
	store, err := jsonfile.Open(filepath.Join(t.TempDir(), "state.json"))
	require.NoError(t, err)

	ctx := context.Background()
	for _, item := range items {
		require.NoError(t, store.UpsertItem(ctx, item))
	}
	src := &testutil.MockSource{NameVal: "test-src"}
	runner := syncp.New(store, []core.Source{src})
	return actions.New(store, runner), store
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

// --- DismissItem ---

func TestDismissItem_SetsDismissed(t *testing.T) {
	item := makeItem("github:owner/repo#1")
	h, store := newActions(t, item)
	ctx := context.Background()

	require.NoError(t, h.DismissItem(ctx, item.ID))

	state, err := store.GetItemState(ctx, item.ID)
	require.NoError(t, err)
	require.NotNil(t, state)
	assert.True(t, state.Dismissed)
}

// DismissItem must merge into existing state, not replace it entirely.
func TestDismissItem_PreservesExistingState(t *testing.T) {
	item := makeItem("github:owner/repo#1")
	h, store := newActions(t, item)
	ctx := context.Background()

	future := time.Now().Add(time.Hour)
	store.SetItemState(ctx, core.ItemState{ItemID: item.ID, SnoozedUntil: &future})

	require.NoError(t, h.DismissItem(ctx, item.ID))

	state, err := store.GetItemState(ctx, item.ID)
	require.NoError(t, err)
	require.NotNil(t, state)
	assert.True(t, state.Dismissed)
	assert.Equal(t, future, *state.SnoozedUntil)
}

func TestDismissItem_ErrNotFound(t *testing.T) {
	h, _ := newActions(t) // empty store
	err := h.DismissItem(context.Background(), "github:owner/repo#999")
	assert.ErrorIs(t, err, actions.ErrNotFound)
}

// --- SnoozeItem ---

func TestSnoozeItem_SetsSnoozedUntil(t *testing.T) {
	item := makeItem("github:owner/repo#2")
	h, store := newActions(t, item)
	ctx := context.Background()

	until := time.Now().Add(24 * time.Hour).Truncate(time.Second)
	require.NoError(t, h.SnoozeItem(ctx, item.ID, until))

	state, err := store.GetItemState(ctx, item.ID)
	require.NoError(t, err)
	require.NotNil(t, state)
	require.NotNil(t, state.SnoozedUntil)
	assert.Equal(t, until, *state.SnoozedUntil)
}

// SnoozeItem must merge into existing state, not replace it entirely.
func TestSnoozeItem_PreservesExistingState(t *testing.T) {
	item := makeItem("github:owner/repo#2")
	h, store := newActions(t, item)
	ctx := context.Background()

	store.SetItemState(ctx, core.ItemState{ItemID: item.ID, Dismissed: true})

	require.NoError(t, h.SnoozeItem(ctx, item.ID, time.Now().Add(time.Hour)))

	state, err := store.GetItemState(ctx, item.ID)
	require.NoError(t, err)
	require.NotNil(t, state)
	assert.True(t, state.Dismissed)
}

func TestSnoozeItem_ErrNotFound(t *testing.T) {
	h, _ := newActions(t)
	err := h.SnoozeItem(context.Background(), "github:owner/repo#999", time.Now().Add(time.Hour))
	assert.ErrorIs(t, err, actions.ErrNotFound)
}

// --- ListItems ---

func TestListItems_NoFilter_ReturnsAll(t *testing.T) {
	items := []core.Item{makeItem("github:owner/repo#1"), makeItem("github:owner/repo#2")}
	h, _ := newActions(t, items...)

	got, err := h.ListItems(context.Background(), core.ItemFilter{})
	require.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestListItems_ProjectsFields(t *testing.T) {
	item := core.Item{
		ID:         "github:owner/repo#1",
		Source:     "github",
		Type:       "pr",
		Title:      "Fix the thing",
		URL:        "https://github.com/owner/repo/pull/1",
		Namespace:  "owner/repo",
		UpdatedAt:  time.Now(),
		WaitsOnMe:  true,
		IsAssigned: true,
	}
	h, _ := newActions(t, item)

	got, err := h.ListItems(context.Background(), core.ItemFilter{})
	require.NoError(t, err)
	require.Len(t, got, 1)

	s := got[0]
	assert.Equal(t, item.ID, s.ID)
	assert.Equal(t, item.Title, s.Title)
	assert.Equal(t, item.URL, s.URL)
	assert.True(t, s.IsAssigned)
}

func TestListItems_NeedsAttentionFilter_ExcludesDismissed(t *testing.T) {
	item := makeItem("github:owner/repo#1")
	h, store := newActions(t, item)
	ctx := context.Background()

	store.SetItemState(ctx, core.ItemState{ItemID: item.ID, Dismissed: true})

	got, err := h.ListItems(ctx, core.ItemFilter{NeedsAttention: true})
	require.NoError(t, err)
	assert.Empty(t, got)
}

// --- Summary ---

func TestSummary_OnlyReturnsItemsNeedingAttention(t *testing.T) {
	needsAttention := makeItem("github:owner/repo#1")
	dismissed := makeItem("github:owner/repo#2")

	h, store := newActions(t, needsAttention, dismissed)
	ctx := context.Background()

	store.SetItemState(ctx, core.ItemState{ItemID: dismissed.ID, Dismissed: true})

	got, err := h.Summary(ctx)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, needsAttention.ID, got[0].ID)
}

// --- SyncSource ---

func TestSyncSource_UnknownName_ReturnsError(t *testing.T) {
	h, _ := newActions(t)
	err := h.SyncSource(context.Background(), "does-not-exist")
	assert.Error(t, err)
}

func TestSyncSource_KnownName_CallsSource(t *testing.T) {
	store, err := jsonfile.Open(filepath.Join(t.TempDir(), "state.json"))
	require.NoError(t, err)

	src := &testutil.MockSource{NameVal: "my-src"}
	runner := syncp.New(store, []core.Source{src})
	h := actions.New(store, runner)

	require.NoError(t, h.SyncSource(context.Background(), "my-src"))
	assert.Equal(t, 1, src.SyncCalled)
}
