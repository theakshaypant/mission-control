package core

import (
	"encoding/json"
	"time"
)

// SourceKind is a string identifier for a source type (e.g. "github", "jira").
// Each source package defines its own Kind constant typed as SourceKind.
type SourceKind string

// ItemType is a string identifier for an item's type within its source
// (e.g. "pr", "issue", "ticket"). Each source package defines its own
// type constants typed as ItemType.
type ItemType string

// Item is the unified representation of a work item across all sources.
type Item struct {
	ID     string
	Source SourceKind
	Type   ItemType

	Title string
	URL   string

	// Namespace is the source-specific grouping context:
	// "org/repo" for GitHub, project key for Jira, channel name for Slack, etc.
	Namespace string

	CreatedAt time.Time
	UpdatedAt time.Time

	// WaitsOnMe indicates that the user is expected to take an action on this item —
	// e.g. a review request, a direct mention, an assigned ticket awaiting response.
	// Each source is responsible for mapping its own concepts onto this signal.
	WaitsOnMe bool

	// IsAssigned indicates the item is directly assigned to the user.
	IsAssigned bool

	// UserActivityAt is the most recent time the authenticated user took a
	// configured interaction action on this item (e.g. reviewed, commented).
	// Set by the source during sync; nil means no recorded user activity.
	// What counts as an interaction is defined per-source in Config.
	UserActivityAt *time.Time

	// ExternalRefs holds cross-source references found in item content,
	// e.g. a Jira ticket ID mentioned in a GitHub PR description.
	ExternalRefs []string

	// Attributes holds source-specific typed data serialized as JSON.
	// Each source defines its own attributes struct and marshals it here.
	Attributes json.RawMessage
}
