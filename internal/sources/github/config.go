package github

import (
	"fmt"
	"strings"
)

// Interaction is a GitHub activity type that can be configured to count
// as the user having interacted with an item.
type Interaction string

const (
	InteractionReview         Interaction = "review"
	InteractionComment        Interaction = "comment"
	InteractionApprove        Interaction = "approve"
	InteractionRequestChanges Interaction = "request_changes"
)

// FetchScope controls which PRs are retrieved for each configured repo.
type FetchScope string

const (
	// FetchScopeInvolved fetches only PRs where the user is author, reviewer,
	// or assignee, using GitHub's search API across all configured repos.
	FetchScopeInvolved FetchScope = "involved"
	// FetchScopeAll fetches all open PRs in each configured repo.
	FetchScopeAll FetchScope = "all"
)

// WaitsOnMeSignal is a condition that marks a PR as needing the user's attention.
type WaitsOnMeSignal string

const (
	// WaitsOnMeUnreviewed fires when the user has never reviewed or commented
	// on a PR they did not author.
	WaitsOnMeUnreviewed WaitsOnMeSignal = "unreviewed"
	// WaitsOnMeAuthorUpdated fires when the user has previously reviewed or
	// commented, and the PR author has since pushed new commits or replied.
	WaitsOnMeAuthorUpdated WaitsOnMeSignal = "author_updated"
	// WaitsOnMePeerActivity fires when someone who is neither the PR author
	// nor the user has reviewed or commented since the user's last activity
	// (or at any time if the user has never engaged).
	WaitsOnMePeerActivity WaitsOnMeSignal = "peer_activity"
	// WaitsOnMeApprovedNotMerged fires when GitHub considers the PR approved
	// (all required reviews satisfied) but it has not yet been merged.
	WaitsOnMeApprovedNotMerged WaitsOnMeSignal = "approved_not_merged"
	// WaitsOnMeReviewReceived fires when the user is the PR author and someone
	// else has commented or reviewed since the user's last commit push or comment.
	WaitsOnMeReviewReceived WaitsOnMeSignal = "review_received"
	// WaitsOnMeApproved fires when the user is the PR author and GitHub
	// considers the PR approved.
	WaitsOnMeApproved WaitsOnMeSignal = "approved"
	// WaitsOnMeStale fires when the PR has had no activity for longer than
	// the configured StaleDays threshold (default 30 days).
	WaitsOnMeStale WaitsOnMeSignal = "stale"
)

// AssignedSignal controls which conditions mark a PR as assigned to the user.
type AssignedSignal string

const (
	// AssignedSignalAssignee marks a PR as assigned if the user is in its
	// explicit assignees list.
	AssignedSignalAssignee AssignedSignal = "assignee"
	// AssignedSignalAuthor marks a PR as assigned if the user authored it.
	AssignedSignalAuthor AssignedSignal = "author"
	// AssignedSignalReviewer marks a PR as assigned if the user has been
	// explicitly requested as a reviewer.
	AssignedSignalReviewer AssignedSignal = "reviewer"
)

// Config holds configuration for a single GitHub source instance.
// A user may define multiple GitHub sources (e.g. work and personal accounts,
// or repos with different fetch scopes).
type Config struct {
	Token string `yaml:"token"`
	// GitHub login, used to detect user activity
	User string `yaml:"user"`
	// each entry must be in "owner/repo" format
	Repos []string `yaml:"repos"`

	// Host is the GitHub hostname. Leave empty for github.com.
	// For GitHub Enterprise Server, set this to your instance hostname,
	// e.g. "github.mycompany.com". The GraphQL endpoint is derived automatically:
	// github.com → https://api.github.com/graphql
	// <host>     → https://<host>/api/graphql
	Host string `yaml:"host"`

	// Interactions lists the activity types that count as the user having
	// interacted with an item. Defaults to all interactions if empty.
	// Valid values: "review", "comment", "approve", "request_changes"
	Interactions []Interaction `yaml:"interactions"`

	// PRScope controls which PRs are fetched. "involved" (default) uses
	// GitHub search to return only PRs the user is involved in; "all"
	// fetches every open PR in each configured repo.
	PRScope FetchScope `yaml:"pr_scope"`

	// MaxPRs is the maximum number of PRs to fetch per repo (scope "all")
	// or in total across repos (scope "involved"). Defaults to 50.
	MaxPRs int `yaml:"max_prs"`

	// WaitsOnMe lists signals that mark a PR as needing the user's attention.
	// Only PRs where at least one signal fires are returned.
	// Defaults to [unreviewed, author_updated, peer_activity, review_received].
	// Valid values: "unreviewed", "author_updated", "peer_activity",
	// "approved_not_merged", "review_received", "approved", "stale"
	WaitsOnMe []WaitsOnMeSignal `yaml:"waits_on_me"`

	// StaleDays is the number of days of inactivity after which a PR is
	// considered stale (used by the "stale" signal). Defaults to 30.
	StaleDays int `yaml:"stale_days"`

	// IsAssigned lists conditions that mark a PR as assigned to the user.
	// Defaults to all conditions (author + assignee) if empty.
	// Valid values: "assignee", "author"
	IsAssigned []AssignedSignal `yaml:"is_assigned"`
}

func (c *Config) Validate() error {
	if c.Token == "" {
		return fmt.Errorf("github: token is required")
	}
	if c.User == "" {
		return fmt.Errorf("github: user is required")
	}
	if len(c.Repos) == 0 {
		return fmt.Errorf("github: at least one repo is required")
	}
	for _, r := range c.Repos {
		if !strings.Contains(r, "/") {
			return fmt.Errorf("github: repo %q must be in owner/repo format", r)
		}
	}
	if strings.ContainsAny(c.Host, " /") || strings.HasPrefix(c.Host, "http") {
		return fmt.Errorf("github: host must be a bare hostname (e.g. github.mycompany.com), not a URL")
	}
	for _, i := range c.Interactions {
		switch i {
		case InteractionReview, InteractionComment, InteractionApprove, InteractionRequestChanges:
		default:
			return fmt.Errorf("github: unknown interaction type %q", i)
		}
	}
	switch c.PRScope {
	case "", FetchScopeInvolved, FetchScopeAll:
	default:
		return fmt.Errorf("github: unknown pr_scope %q", c.PRScope)
	}
	for _, s := range c.WaitsOnMe {
		switch s {
		case WaitsOnMeUnreviewed, WaitsOnMeAuthorUpdated, WaitsOnMePeerActivity,
			WaitsOnMeApprovedNotMerged, WaitsOnMeReviewReceived, WaitsOnMeApproved,
			WaitsOnMeStale:
		default:
			return fmt.Errorf("github: unknown waits_on_me signal %q", s)
		}
	}
	if c.StaleDays < 0 {
		return fmt.Errorf("github: stale_days must be non-negative")
	}
	for _, s := range c.IsAssigned {
		switch s {
		case AssignedSignalAssignee, AssignedSignalAuthor, AssignedSignalReviewer:
		default:
			return fmt.Errorf("github: unknown is_assigned signal %q", s)
		}
	}
	return nil
}
