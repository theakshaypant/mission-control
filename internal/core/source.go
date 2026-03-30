package core

import (
	"context"
	"time"
)

// Source is the interface all data sources must implement.
type Source interface {
	// Name returns the user-defined label for this source instance, e.g. "work-github".
	Name() string

	// Kind returns the source type, e.g. SourceGitHub or SourceJira.
	Kind() SourceKind

	// Config returns the configuration for this source instance.
	Config() SourceConfig

	// Sync fetches new or updated items since the last sync.
	// The source is responsible for tracking its own sync cursor.
	Sync(ctx context.Context) ([]Item, error)

	// LastSyncedAt returns the time of the last successful sync, or nil if
	// the source has never synced.
	LastSyncedAt() *time.Time
}

// SourceConfig is implemented by each source's typed config struct.
type SourceConfig interface {
	Validate() error
}
