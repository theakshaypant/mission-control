//go:build integration

package e2e

// Per-signal tests: each test activates exactly one signal so the output is
// unambiguous. Run individually to validate signal logic against the GitHub UI.
// All use pr_scope: all — the recommended mode for maintainers.

import (
	"context"
	"testing"

	github "github.com/theakshaypant/mission-control/internal/sources/github"
)

// TestGitHubPRSignalUnreviewed surfaces PRs you have never reviewed or commented
// on (excluding your own).
//
// Validate: open each listed PR in GitHub and confirm you have left no review
// or comment.
func TestGitHubPRSignalUnreviewed(t *testing.T) {
	token, user, repos := integrationEnv(t)

	cfg := baseConfig(token, user, repos)
	cfg.WaitsOnMe = []github.WaitsOnMeSignal{github.WaitsOnMeUnreviewed}
	src := github.New("e2e-unreviewed", cfg)

	items, err := src.Sync(context.Background(), nil)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}
	printItems(t, items)
}

// TestGitHubPRSignalAuthorUpdated surfaces PRs where you have already reviewed
// or commented, and the author has since pushed new commits or replied.
//
// Validate: my activity should be set, and the PR's updated date should be
// after that timestamp. Open the PR to confirm the author has new activity
// since your last comment.
func TestGitHubPRSignalAuthorUpdated(t *testing.T) {
	token, user, repos := integrationEnv(t)

	cfg := baseConfig(token, user, repos)
	cfg.WaitsOnMe = []github.WaitsOnMeSignal{github.WaitsOnMeAuthorUpdated}
	src := github.New("e2e-author-updated", cfg)

	items, err := src.Sync(context.Background(), nil)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}
	printItems(t, items)
}

// TestGitHubPRSignalPeerActivity surfaces PRs where someone who is neither the
// author nor you has reviewed or commented since your last activity.
//
// Validate: every listed PR should have a review or comment from a third party
// (not the author, not you) that is newer than your last review or comment.
// If you've never engaged with the PR, any third-party activity qualifies.
func TestGitHubPRSignalPeerActivity(t *testing.T) {
	token, user, repos := integrationEnv(t)

	cfg := baseConfig(token, user, repos)
	cfg.WaitsOnMe = []github.WaitsOnMeSignal{github.WaitsOnMePeerActivity}
	src := github.New("e2e-peer-activity", cfg)

	items, err := src.Sync(context.Background(), nil)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}
	printItems(t, items)
}

// TestGitHubPRSignalReviewReceived surfaces your open PRs where someone else has
// commented or reviewed since your last commit push or comment.
//
// Validate: every listed PR should show your login as author, and have a
// review or comment from someone else that postdates your most recent push
// or comment.
func TestGitHubPRSignalReviewReceived(t *testing.T) {
	token, user, repos := integrationEnv(t)

	cfg := baseConfig(token, user, repos)
	cfg.WaitsOnMe = []github.WaitsOnMeSignal{github.WaitsOnMeReviewReceived}
	src := github.New("e2e-review-received", cfg)

	items, err := src.Sync(context.Background(), nil)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}
	printItems(t, items)
}

// TestGitHubPRSignalApprovedNotMerged surfaces PRs that GitHub considers fully
// approved but that haven't been merged yet.
//
// Validate: every listed PR should show "Approved" in the GitHub UI with no
// pending change requests, and still be open.
func TestGitHubPRSignalApprovedNotMerged(t *testing.T) {
	token, user, repos := integrationEnv(t)

	cfg := baseConfig(token, user, repos)
	cfg.WaitsOnMe = []github.WaitsOnMeSignal{github.WaitsOnMeApprovedNotMerged}
	src := github.New("e2e-approved-not-merged", cfg)

	items, err := src.Sync(context.Background(), nil)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}
	printItems(t, items)
}

// TestGitHubPRSignalApproved surfaces your open PRs that GitHub considers fully
// approved — ready to merge or act on.
//
// Validate: every listed PR should be one you authored and should show
// "Approved" in the GitHub UI with no pending change requests.
func TestGitHubPRSignalApproved(t *testing.T) {
	token, user, repos := integrationEnv(t)

	cfg := baseConfig(token, user, repos)
	cfg.WaitsOnMe = []github.WaitsOnMeSignal{github.WaitsOnMeApproved}
	src := github.New("e2e-approved", cfg)

	items, err := src.Sync(context.Background(), nil)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}
	printItems(t, items)
}

// TestGitHubPRSignalStale surfaces PRs with no activity for longer than
// stale_days (default 30 days).
//
// Validate: every listed PR's updated date should be more than 30 days ago.
func TestGitHubPRSignalStale(t *testing.T) {
	token, user, repos := integrationEnv(t)

	cfg := baseConfig(token, user, repos)
	cfg.WaitsOnMe = []github.WaitsOnMeSignal{github.WaitsOnMeStale}
	cfg.StaleDays = 30
	src := github.New("e2e-stale", cfg)

	items, err := src.Sync(context.Background(), nil)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}
	printItems(t, items)
}

// TestGitHubPRSignalAll runs every signal in a separate subtest so you can
// compare their outputs in one go. Useful for a full audit across your repos.
func TestGitHubPRSignalAll(t *testing.T) {
	token, user, repos := integrationEnv(t)

	signals := []github.WaitsOnMeSignal{
		github.WaitsOnMeUnreviewed,
		github.WaitsOnMeAuthorUpdated,
		github.WaitsOnMePeerActivity,
		github.WaitsOnMeApprovedNotMerged,
		github.WaitsOnMeReviewReceived,
		github.WaitsOnMeApproved,
		github.WaitsOnMeStale,
	}

	for _, sig := range signals {
		t.Run(string(sig), func(t *testing.T) {
			cfg := baseConfig(token, user, repos)
			cfg.WaitsOnMe = []github.WaitsOnMeSignal{sig}
			src := github.New("e2e-signal", cfg)

			items, err := src.Sync(context.Background(), nil)
			if err != nil {
				t.Fatalf("Sync failed: %v", err)
			}
			printItems(t, items)
		})
	}
}
