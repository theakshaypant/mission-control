//go:build integration

package e2e

import (
	"context"
	"testing"

	github "github.com/theakshaypant/mission-control/internal/sources/github"
)

// TestGitHubIssueScopeInvolved confirms that the involved scope returns only
// open issues where the user is author, assignee, or mentioned.
//
// Validate: every listed issue should be one you are involved in on GitHub.
func TestGitHubIssueScopeInvolved(t *testing.T) {
	token, user, repos := integrationEnv(t)

	cfg := baseIssueConfig(token, user, repos)
	src := github.New("e2e-issues-involved", cfg)

	items, err := src.Sync(context.Background())
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	for _, item := range items {
		if item.Type != github.TypeIssue {
			t.Errorf("expected only issues, got type %q for %s", item.Type, item.ID)
		}
	}

	printItems(t, items)
}

// TestGitHubIssueScopeAll confirms that fetching all open issues in a repo
// works end-to-end and that MaxIssues is respected as a per-repo cap.
//
// Validate: the item count should not exceed limit × number of repos.
func TestGitHubIssueScopeAll(t *testing.T) {
	token, user, repos := integrationEnv(t)

	const limit = 10
	cfg := baseIssueConfig(token, user, repos)
	cfg.IssueScope = github.FetchScopeAll
	cfg.MaxIssues = limit
	src := github.New("e2e-issues-all", cfg)

	items, err := src.Sync(context.Background())
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}
	if len(items) > limit*len(repos) {
		t.Errorf("got %d items, expected at most %d (limit %d × %d repos)",
			len(items), limit*len(repos), limit, len(repos))
	}

	printItems(t, items)
}

// TestGitHubIssueSignalUnreviewed surfaces open issues the user is involved in
// but has never commented on.
//
// Validate: every listed issue should have no comment from you on GitHub.
func TestGitHubIssueSignalUnreviewed(t *testing.T) {
	token, user, repos := integrationEnv(t)

	cfg := baseIssueConfig(token, user, repos)
	cfg.WaitsOnMe = []github.WaitsOnMeSignal{github.WaitsOnMeUnreviewed}
	src := github.New("e2e-issues-unreviewed", cfg)

	items, err := src.Sync(context.Background())
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	for _, item := range items {
		if item.UserActivityAt != nil {
			t.Errorf("unreviewed signal fired but UserActivityAt is set for %s", item.ID)
		}
	}

	printItems(t, items)
}

// TestGitHubIssueSignalAuthorUpdated surfaces issues where the user has
// commented and the author has since replied.
//
// Validate: every listed issue should show "my activity" set and the author
// should have a more recent comment than your last one.
func TestGitHubIssueSignalAuthorUpdated(t *testing.T) {
	token, user, repos := integrationEnv(t)

	cfg := baseIssueConfig(token, user, repos)
	cfg.WaitsOnMe = []github.WaitsOnMeSignal{github.WaitsOnMeAuthorUpdated}
	src := github.New("e2e-issues-author-updated", cfg)

	items, err := src.Sync(context.Background())
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	printItems(t, items)
}

// TestGitHubIssueSignalPeerActivity surfaces issues where a third party (neither
// the author nor the user) has commented since the user's last activity.
//
// Validate: every listed issue should have a comment from someone other than
// you and the author, newer than your last comment.
func TestGitHubIssueSignalPeerActivity(t *testing.T) {
	token, user, repos := integrationEnv(t)

	cfg := baseIssueConfig(token, user, repos)
	cfg.WaitsOnMe = []github.WaitsOnMeSignal{github.WaitsOnMePeerActivity}
	src := github.New("e2e-issues-peer-activity", cfg)

	items, err := src.Sync(context.Background())
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	printItems(t, items)
}

// TestGitHubIssueSignalReviewReceived surfaces issues the user opened where
// someone else has commented since the user's last comment.
//
// Validate: every listed issue should be authored by you and have a comment
// from someone else newer than your last comment.
func TestGitHubIssueSignalReviewReceived(t *testing.T) {
	token, user, repos := integrationEnv(t)

	cfg := baseIssueConfig(token, user, repos)
	cfg.WaitsOnMe = []github.WaitsOnMeSignal{github.WaitsOnMeReviewReceived}
	src := github.New("e2e-issues-review-received", cfg)

	items, err := src.Sync(context.Background())
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	printItems(t, items)
}

// TestGitHubIssueSignalStale surfaces open issues with no activity for longer
// than the configured stale threshold.
//
// Validate: every listed issue's "updated" timestamp should be older than
// stale_days days.
func TestGitHubIssueSignalStale(t *testing.T) {
	token, user, repos := integrationEnv(t)

	cfg := baseIssueConfig(token, user, repos)
	cfg.WaitsOnMe = []github.WaitsOnMeSignal{github.WaitsOnMeStale}
	cfg.StaleDays = 30
	src := github.New("e2e-issues-stale", cfg)

	items, err := src.Sync(context.Background())
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	printItems(t, items)
}

// TestGitHubIssueSignalAll runs every applicable issue signal in a separate
// subtest so you can compare their outputs in one go.
func TestGitHubIssueSignalAll(t *testing.T) {
	token, user, repos := integrationEnv(t)

	signals := []github.WaitsOnMeSignal{
		github.WaitsOnMeUnreviewed,
		github.WaitsOnMeAuthorUpdated,
		github.WaitsOnMePeerActivity,
		github.WaitsOnMeReviewReceived,
		github.WaitsOnMeStale,
	}

	for _, sig := range signals {
		t.Run(string(sig), func(t *testing.T) {
			cfg := baseIssueConfig(token, user, repos)
			cfg.WaitsOnMe = []github.WaitsOnMeSignal{sig}
			src := github.New("e2e-issues-signal", cfg)

			items, err := src.Sync(context.Background())
			if err != nil {
				t.Fatalf("Sync failed: %v", err)
			}
			printItems(t, items)
		})
	}
}

// TestGitHubMixedSync confirms that when both PRScope and IssueScope are set,
// a single Sync returns both PRs and issues.
//
// Validate: the result should contain items with both "pr" and "issue" types.
func TestGitHubMixedSync(t *testing.T) {
	token, user, repos := integrationEnv(t)

	cfg := baseConfig(token, user, repos)
	cfg.IssueScope = github.FetchScopeInvolved
	cfg.MaxIssues = 50
	src := github.New("e2e-issues-mixed", cfg)

	items, err := src.Sync(context.Background())
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	prCount, issueCount := 0, 0
	for _, item := range items {
		switch item.Type {
		case github.TypePR:
			prCount++
		case github.TypeIssue:
			issueCount++
		}
	}
	t.Logf("mixed sync: %d PRs, %d issues", prCount, issueCount)

	printItems(t, items)
}

// TestGitHubIssueIncrementalSync verifies that a second sync immediately after
// the first returns no issue items.
func TestGitHubIssueIncrementalSync(t *testing.T) {
	token, user, repos := integrationEnv(t)

	src := github.New("e2e-issues-incremental", baseIssueConfig(token, user, repos))

	first, err := src.Sync(context.Background())
	if err != nil {
		t.Fatalf("first Sync failed: %v", err)
	}
	t.Logf("first sync: %d items", len(first))

	second, err := src.Sync(context.Background())
	if err != nil {
		t.Fatalf("second Sync failed: %v", err)
	}
	t.Logf("second sync: %d items (expected 0)", len(second))

	if len(second) != 0 {
		t.Errorf("expected 0 items on incremental sync, got %d", len(second))
		for _, item := range second {
			t.Logf("  unexpected: %s  updated=%s", item.ID, item.UpdatedAt.Format("2006-01-02T15:04:05Z"))
		}
	}
}
