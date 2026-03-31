package actions

import (
	"context"
	"fmt"
)

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
