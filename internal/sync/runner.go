// Package sync provides the orchestration layer that wires sources to the store.
package sync

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/theakshaypant/mission-control/internal/core"
)

// Runner orchestrates syncing all configured sources into the store.
type Runner struct {
	store   core.Store
	sources []core.Source
}

// New returns a Runner that will sync the given sources into store.
func New(store core.Store, sources []core.Source) *Runner {
	return &Runner{store: store, sources: sources}
}

// Sources returns the names of all configured sources.
func (r *Runner) Sources() []string {
	names := make([]string, len(r.sources))
	for i, src := range r.sources {
		names[i] = src.Name()
	}
	return names
}

// fetchResult holds the outcome of one source's network fetch.
type fetchResult struct {
	src   core.Source
	items []core.Item
	err   error
}

// Sync syncs a single source by name. Returns an error if the name is not found.
func (r *Runner) Sync(ctx context.Context, name string) error {
	for _, src := range r.sources {
		if src.Name() == name {
			res := r.fetch(ctx, src)
			if res.err != nil {
				return fmt.Errorf("sync %q: %w", name, res.err)
			}
			return r.writeResults(ctx, res)
		}
	}
	return fmt.Errorf("sync: unknown source %q", name)
}

// SyncAll syncs all sources. Network fetches run concurrently; store writes
// are sequential to avoid contention on the current JSON file backend.
//
// For each source it:
//  1. Loads the last sync cursor from the store.
//  2. Calls Source.Sync concurrently with that cursor (nil on first run → full fetch).
//  3. Upserts each returned item into the store.
//  4. Advances ItemState.LastInteractedAt when the source reports user activity.
//  5. Records the sync time so the next run can use it as a cursor.
//
// All fetches are attempted even if one fails; the first error is returned after
// all writes for successful sources are committed.
func (r *Runner) SyncAll(ctx context.Context) error {
	results := r.fetchAll(ctx)

	var firstErr error
	for _, res := range results {
		if res.err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("sync %q: %w", res.src.Name(), res.err)
			}
			continue
		}
		if err := r.writeResults(ctx, res); err != nil {
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

// fetch loads the cursor and calls Sync for a single source.
func (r *Runner) fetch(ctx context.Context, src core.Source) fetchResult {
	since, err := r.store.GetLastSyncedAt(ctx, src.Name())
	if err != nil {
		return fetchResult{src: src, err: fmt.Errorf("load cursor: %w", err)}
	}
	items, err := src.Sync(ctx, since)
	if err != nil {
		return fetchResult{src: src, err: fmt.Errorf("fetch: %w", err)}
	}
	return fetchResult{src: src, items: items}
}

// fetchAll runs all source fetches concurrently and returns results in the
// same order as r.sources.
func (r *Runner) fetchAll(ctx context.Context) []fetchResult {
	results := make([]fetchResult, len(r.sources))
	var wg sync.WaitGroup

	for i, src := range r.sources {
		wg.Add(1)
		go func(i int, src core.Source) {
			defer wg.Done()
			results[i] = r.fetch(ctx, src)
		}(i, src)
	}

	wg.Wait()
	return results
}

// writeResults commits one source's fetch results to the store.
func (r *Runner) writeResults(ctx context.Context, res fetchResult) error {
	for _, item := range res.items {
		// Closed items are tombstones: remove any stale entry from the store.
		if item.Closed {
			if err := r.store.DeleteItem(ctx, item.ID); err != nil {
				return fmt.Errorf("sync %q: delete closed %s: %w", res.src.Name(), item.ID, err)
			}
			continue
		}
		item.SourceName = res.src.Name()
		if err := r.store.UpsertItem(ctx, item); err != nil {
			return fmt.Errorf("sync %q: upsert %s: %w", res.src.Name(), item.ID, err)
		}
		if item.UserActivityAt != nil {
			if err := r.advanceLastInteracted(ctx, item.ID, item.UserActivityAt); err != nil {
				return fmt.Errorf("sync %q: %w", res.src.Name(), err)
			}
		}
	}

	now := time.Now()
	if err := r.store.SetLastSyncedAt(ctx, res.src.Name(), now); err != nil {
		return fmt.Errorf("sync %q: record cursor: %w", res.src.Name(), err)
	}
	return nil
}

// advanceLastInteracted sets ItemState.LastInteractedAt to t if t is more
// recent than the currently stored value (or if no state exists yet).
func (r *Runner) advanceLastInteracted(ctx context.Context, itemID string, t *time.Time) error {
	state, err := r.store.GetItemState(ctx, itemID)
	if err != nil {
		return fmt.Errorf("get state %s: %w", itemID, err)
	}
	if state == nil {
		state = &core.ItemState{ItemID: itemID}
	}
	if state.LastInteractedAt == nil || t.After(*state.LastInteractedAt) {
		state.LastInteractedAt = t
		if err := r.store.SetItemState(ctx, *state); err != nil {
			return fmt.Errorf("set state %s: %w", itemID, err)
		}
	}
	return nil
}
