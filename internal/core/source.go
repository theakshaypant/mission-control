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

	// Sync fetches new or updated items. since is the cursor from the last
	// successful sync; pass nil for a full fetch on first run.
	Sync(ctx context.Context, since *time.Time) ([]Item, error)
}

// SourceConfig is implemented by each source's typed config struct.
type SourceConfig interface {
	Validate() error
}
