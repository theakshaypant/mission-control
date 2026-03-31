package actions

import (
	"context"
	"fmt"
	"time"
)

// SourceStatus reports when a source last successfully synced.
type SourceStatus struct {
	Name         string     `json:"name"`
	LastSyncedAt *time.Time `json:"last_synced_at"`
}

// SyncAll triggers a full sync of all configured sources.
func (a *Actions) SyncAll(ctx context.Context) error {
	if err := a.runner.SyncAll(ctx); err != nil {
		return fmt.Errorf("sync all: %w", err)
	}
	return nil
}

// SyncSource triggers a sync for a single named source.
// Returns an error if the source name is not found.
func (a *Actions) SyncSource(ctx context.Context, name string) error {
	if err := a.runner.Sync(ctx, name); err != nil {
		return fmt.Errorf("sync %q: %w", name, err)
	}
	return nil
}

// SyncStatus returns the last successful sync time for each configured source.
func (a *Actions) SyncStatus(ctx context.Context) ([]SourceStatus, error) {
	names := a.runner.Sources()
	statuses := make([]SourceStatus, 0, len(names))
	for _, name := range names {
		t, err := a.store.GetLastSyncedAt(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("sync status %s: %w", name, err)
		}
		statuses = append(statuses, SourceStatus{Name: name, LastSyncedAt: t})
	}
	return statuses, nil
}
