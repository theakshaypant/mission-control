//go:build integration

package e2e

import (
	"context"
	"testing"
	"time"

	github "github.com/theakshaypant/mission-control/internal/sources/github"
)

// TestGitHubPRScopeAll confirms that fetching all open PRs in a repo works
// end-to-end and that MaxPRs is respected as a per-repo cap.
//
// Validate: the item count should not exceed limit × number of repos.
func TestGitHubPRScopeAll(t *testing.T) {
	token, user, repos := integrationEnv(t)

	const limit = 10
	cfg := baseConfig(token, user, repos)
	cfg.MaxPRs = limit
	src := github.New("e2e-fetch-all", cfg)

	items, err := src.Sync(context.Background(), nil)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}
	if len(items) > limit*len(repos) {
		t.Errorf("got %d items, expected at most %d (limit %d × %d repos)",
			len(items), limit*len(repos), limit, len(repos))
	}
	printItems(t, items)
}

// TestGitHubPRScopeInvolved confirms that the involved scope fetches only PRs
// where the user is author, reviewer, or assignee.
//
// Validate: every listed PR should have your login as author, assignee, or
// requested reviewer.
func TestGitHubPRScopeInvolved(t *testing.T) {
	token, user, repos := integrationEnv(t)

	cfg := baseConfig(token, user, repos)
	cfg.PRScope = github.FetchScopeInvolved
	cfg.MaxPRs = 50
	src := github.New("e2e-fetch-involved", cfg)

	items, err := src.Sync(context.Background(), nil)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	printItems(t, items)
}

// TestGitHubPRAssignedAsReviewer surfaces PRs where you are an explicit requested
// reviewer, using the unreviewed signal to keep the result set focused.
//
// Validate: every listed PR should show "assigned to me: true" and you should
// appear in the "Reviewers" section of the PR on GitHub.
func TestGitHubPRAssignedAsReviewer(t *testing.T) {
	token, user, repos := integrationEnv(t)

	cfg := baseConfig(token, user, repos)
	cfg.IsAssigned = []github.AssignedSignal{github.AssignedSignalReviewer}
	cfg.WaitsOnMe = []github.WaitsOnMeSignal{github.WaitsOnMeUnreviewed}
	src := github.New("e2e-assigned-reviewer", cfg)

	items, err := src.Sync(context.Background(), nil)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}
	printItems(t, items)
}

// TestGitHubPRIncrementalSync verifies that a second sync immediately after the
// first returns no items — all PRs predate lastSyncedAt so the since-filter
// skips them.
func TestGitHubPRIncrementalSync(t *testing.T) {
	token, user, repos := integrationEnv(t)

	src := github.New("e2e-incremental", baseConfig(token, user, repos))

	now := time.Now()
	first, err := src.Sync(context.Background(), nil)
	if err != nil {
		t.Fatalf("first Sync failed: %v", err)
	}
	t.Logf("first sync:  %d items", len(first))

	second, err := src.Sync(context.Background(), &now)
	if err != nil {
		t.Fatalf("second Sync failed: %v", err)
	}
	t.Logf("second sync: %d items (expected 0 — all PRs predate since)", len(second))

	if len(second) != 0 {
		t.Errorf("expected 0 items on incremental sync, got %d", len(second))
		for _, item := range second {
			t.Logf("  unexpected: %s  updated=%s", item.ID, item.UpdatedAt.Format("2006-01-02T15:04:05Z"))
		}
	}
}
