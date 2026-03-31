//go:build integration

package e2e

// Real-world scenario tests. These mirror the example configurations from the
// docs and are the closest thing to a production end-to-end test.

import (
	"context"
	"testing"

	github "github.com/theakshaypant/mission-control/internal/sources/github"
)

// TestGitHubScenarioMaintainer runs the recommended config for an OSS maintainer:
// fetch all open PRs, all four default signals active. Every item in the output
// includes a "why" line showing which signal(s) fired.
//
// Validate: open each PR and confirm the listed signal(s) describe what you see.
func TestGitHubScenarioMaintainer(t *testing.T) {
	token, user, repos := integrationEnv(t)

	// Default waits_on_me: unreviewed, author_updated, peer_activity, review_received.
	src := github.New("e2e-maintainer", baseConfig(token, user, repos))

	items, err := src.Sync(context.Background(), nil)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}
	printItems(t, items)
}

// TestGitHubScenarioAspiringContributor runs the recommended config for someone
// exploring repos to contribute to: issues only, PRs disabled entirely.
//
// Validate: the result should contain only issue items that you are involved in.
func TestGitHubScenarioAspiringContributor(t *testing.T) {
	token, user, repos := integrationEnv(t)

	cfg := &github.Config{
		Token:      token,
		User:       user,
		Repos:      repos,
		PRScope:    github.FetchScopeNone,
		IssueScope: github.FetchScopeInvolved,
		MaxIssues:  50,
		WaitsOnMe: []github.WaitsOnMeSignal{
			github.WaitsOnMeUnreviewed,
			github.WaitsOnMeAuthorUpdated,
			github.WaitsOnMeReviewReceived,
		},
	}
	src := github.New("e2e-aspiring-contributor", cfg)

	items, err := src.Sync(context.Background(), nil)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}
	for _, item := range items {
		if item.Type != github.TypeIssue {
			t.Errorf("expected only issues with pr_scope:none, got type %q for %s", item.Type, item.ID)
		}
	}
	printItems(t, items)
}

// TestGitHubScenarioContributor runs the recommended config for a contributor
// who only wants to act on PRs they've been tagged on or authored.
//
// Validate: every listed PR should have your login as author, assignee, or
// requested reviewer (GitHub's "involves" filter).
func TestGitHubScenarioContributor(t *testing.T) {
	token, user, repos := integrationEnv(t)

	cfg := &github.Config{
		Token:   token,
		User:    user,
		Repos:   repos,
		PRScope: github.FetchScopeInvolved,
		MaxPRs:  50,
		WaitsOnMe: []github.WaitsOnMeSignal{
			github.WaitsOnMeUnreviewed,
			github.WaitsOnMeAuthorUpdated,
			github.WaitsOnMeReviewReceived,
		},
	}
	src := github.New("e2e-contributor", cfg)

	items, err := src.Sync(context.Background(), nil)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}
	printItems(t, items)
}
