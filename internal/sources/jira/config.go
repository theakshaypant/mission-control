package jira

import (
	"fmt"
	"strings"
)

// WaitsOnMeSignal is a condition that marks a Jira ticket as needing the
// user's attention. At least one signal must fire for a ticket to surface
// in the summary.
type WaitsOnMeSignal string

const (
	// WaitsOnMeAssigned fires when the ticket is assigned to the configured user.
	WaitsOnMeAssigned WaitsOnMeSignal = "assigned"

	// WaitsOnMeCommentReceived fires when someone other than the user has
	// commented on the ticket more recently than the user's last comment.
	// Requires the user to be either the assignee or the reporter of the ticket.
	WaitsOnMeCommentReceived WaitsOnMeSignal = "comment_received"

	// WaitsOnMeStale fires when the ticket has had no activity for longer than
	// the configured StaleDays threshold.
	WaitsOnMeStale WaitsOnMeSignal = "stale"

	// WaitsOnMeStatusChanged fires when the ticket's status changed since the
	// last successful sync. Only active on incremental syncs — skipped on the
	// first (full) sync to avoid surfacing every ticket in the store.
	WaitsOnMeStatusChanged WaitsOnMeSignal = "status_changed"
)

// Interaction is an activity type that counts as the user having acted on a
// ticket, advancing UserActivityAt.
type Interaction string

const (
	// InteractionComment counts a comment posted by the user as an interaction.
	InteractionComment Interaction = "comment"
)

// Board defines a named JQL query whose results are synced as a group.
// The Name is used as Item.Namespace, letting users label boards semantically
// (e.g. "My Open Issues", "Next Release Targets").
type Board struct {
	// Name is the human-readable label for this board, used as Item.Namespace.
	Name string `yaml:"name"`
	// JQL is the Jira Query Language filter that defines which tickets belong to
	// this board. Any valid JQL is accepted; ORDER BY clauses are supported.
	JQL string `yaml:"jql"`
	// MaxResults caps the number of tickets fetched per sync for this board.
	// Defaults to 50.
	MaxResults int `yaml:"max_results"`
}

// Config holds configuration for a single Jira source instance.
// A user may define multiple Jira sources (e.g. for different Jira Cloud sites).
type Config struct {
	// Host is the bare hostname of the Jira Cloud site, e.g. "mycompany.atlassian.net".
	// Do not include "https://" or a trailing slash.
	Host string `yaml:"host"`

	// Email is the user's Atlassian account email address. Used for both
	// Basic authentication and signal evaluation (identifying the user in
	// assignee, reporter, and comment fields).
	Email string `yaml:"email"`

	// Token is the Atlassian API token for Basic authentication.
	// Generate one at https://id.atlassian.com/manage-profile/security/api-tokens.
	Token string `yaml:"token"`

	// APIVersion controls which Jira REST API version to target.
	// Defaults to 3 (Jira Cloud REST API v3). Reserved for future v2 support.
	APIVersion int `yaml:"api_version"`

	// Boards lists the named JQL queries to sync. At least one board is required.
	Boards []Board `yaml:"boards"`

	// WaitsOnMe lists signals that mark a ticket as needing the user's attention.
	// Defaults to [assigned, comment_received, stale, status_changed].
	// Valid values: "assigned", "comment_received", "stale", "status_changed"
	WaitsOnMe []WaitsOnMeSignal `yaml:"waits_on_me"`

	// StaleDays is the number of days of inactivity after which a ticket is
	// considered stale (used by the "stale" signal). Defaults to 14.
	StaleDays int `yaml:"stale_days"`

	// Interactions lists activity types that count as the user having acted on a
	// ticket, advancing UserActivityAt. Defaults to [comment].
	// Valid values: "comment"
	Interactions []Interaction `yaml:"interactions"`

	// DoneStatuses lists workflow status names that are considered terminal.
	// Tickets in these statuses are tombstoned (removed from the store) on sync.
	// Defaults to ["Done", "Closed", "Resolved", "Won't Do"].
	DoneStatuses []string `yaml:"done_statuses"`
}

func (c *Config) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("jira: host is required")
	}
	if strings.ContainsAny(c.Host, " /") || strings.HasPrefix(c.Host, "http") {
		return fmt.Errorf("jira: host must be a bare hostname (e.g. mycompany.atlassian.net), not a URL")
	}
	if c.Email == "" {
		return fmt.Errorf("jira: email is required")
	}
	if c.Token == "" {
		return fmt.Errorf("jira: token is required")
	}
	if len(c.Boards) == 0 {
		return fmt.Errorf("jira: at least one board is required")
	}
	for i, b := range c.Boards {
		if b.Name == "" {
			return fmt.Errorf("jira: board[%d]: name is required", i)
		}
		if b.JQL == "" {
			return fmt.Errorf("jira: board[%d] %q: jql is required", i, b.Name)
		}
		if b.MaxResults < 0 {
			return fmt.Errorf("jira: board[%d] %q: max_results must be non-negative", i, b.Name)
		}
	}
	if c.APIVersion != 0 && c.APIVersion != 3 {
		return fmt.Errorf("jira: api_version %d is not supported (only 3 is currently supported)", c.APIVersion)
	}
	for _, s := range c.WaitsOnMe {
		switch s {
		case WaitsOnMeAssigned, WaitsOnMeCommentReceived, WaitsOnMeStale, WaitsOnMeStatusChanged:
		default:
			return fmt.Errorf("jira: unknown waits_on_me signal %q", s)
		}
	}
	if c.StaleDays < 0 {
		return fmt.Errorf("jira: stale_days must be non-negative")
	}
	for _, i := range c.Interactions {
		switch i {
		case InteractionComment:
		default:
			return fmt.Errorf("jira: unknown interaction %q", i)
		}
	}
	return nil
}
