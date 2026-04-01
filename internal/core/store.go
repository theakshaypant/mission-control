package core

import (
	"context"
	"time"
)

// Store is the persistence interface for items and their state.
// Not yet implemented — the SQLite implementation will satisfy this interface.
type Store interface {
	// UpsertItem inserts or updates an item from a source sync.
	UpsertItem(ctx context.Context, item Item) error

	// DeleteItem removes an item and its state from the store.
	// It is a no-op if the item does not exist.
	DeleteItem(ctx context.Context, id string) error

	// ListItems returns items matching the given filter.
	ListItems(ctx context.Context, filter ItemFilter) ([]Item, error)

	// GetLastSyncedAt returns when a source last successfully synced.
	// Returns nil, nil if the source has never synced.
	GetLastSyncedAt(ctx context.Context, sourceName string) (*time.Time, error)

	// SetLastSyncedAt records the time of a successful source sync.
	SetLastSyncedAt(ctx context.Context, sourceName string, t time.Time) error

	// GetItemState returns the state for a given item ID.
	// Returns nil, nil if no state has been recorded for the item.
	GetItemState(ctx context.Context, itemID string) (*ItemState, error)

	// SetItemState writes the state for an item, replacing any existing state.
	SetItemState(ctx context.Context, state ItemState) error
}

// ItemFilter controls which items are returned by Store.ListItems.
type ItemFilter struct {
	// Source filters to a specific source kind. Zero value means all sources.
	Source SourceKind

	// SourceName filters to a specific named source instance (e.g. "work").
	// Zero value means all source instances.
	SourceName string

	// Types filters to specific item types. Empty means all types.
	Types []ItemType

	// WaitsOnMe filters to items where the user is expected to act.
	WaitsOnMe bool

	// NeedsAttention filters to items that need the user's attention,
	// taking item state (dismissed, snoozed, last interacted) into account.
	NeedsAttention bool

	// Snoozed filters to items that are currently snoozed (snooze time is in the future).
	Snoozed bool
}
