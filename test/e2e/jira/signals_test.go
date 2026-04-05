//go:build integration

package e2e

// Per-signal tests: each test activates exactly one signal so the output is
// unambiguous. Run individually to validate signal logic against the Jira UI.

import (
	"context"
	"testing"
	"time"

	"github.com/theakshaypant/mission-control/internal/sources/jira"
)

// TestJiraSignalAssigned surfaces tickets currently assigned to you.
//
// Validate: every listed ticket should show your email as the assignee in Jira.
// "assigned to me: true" should appear for all items.
func TestJiraSignalAssigned(t *testing.T) {
	token, email, host, jql := integrationEnv(t)

	cfg := baseConfig(token, email, host, jql)
	cfg.WaitsOnMe = []jira.WaitsOnMeSignal{jira.WaitsOnMeAssigned}
	src := jira.New("e2e-assigned", cfg)

	items, err := src.Sync(context.Background(), nil)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}
	printItems(t, items)
}

// TestJiraSignalCommentReceived surfaces tickets where you are the assignee or
// reporter and someone else has commented more recently than your last comment.
//
// Validate: every listed ticket should have a comment from someone other than
// you that is newer than your last comment. Open the ticket to confirm.
func TestJiraSignalCommentReceived(t *testing.T) {
	token, email, host, jql := integrationEnv(t)

	cfg := baseConfig(token, email, host, jql)
	cfg.WaitsOnMe = []jira.WaitsOnMeSignal{jira.WaitsOnMeCommentReceived}
	src := jira.New("e2e-comment-received", cfg)

	items, err := src.Sync(context.Background(), nil)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}
	printItems(t, items)
}

// TestJiraSignalStale surfaces tickets with no activity for longer than
// stale_days (default 14 days).
//
// Validate: every listed ticket's updated date should be more than 14 days ago.
// Cross-check with the "updated" field shown in the Jira list view.
func TestJiraSignalStale(t *testing.T) {
	token, email, host, jql := integrationEnv(t)

	cfg := baseConfig(token, email, host, jql)
	cfg.WaitsOnMe = []jira.WaitsOnMeSignal{jira.WaitsOnMeStale}
	cfg.StaleDays = 14
	src := jira.New("e2e-stale", cfg)

	items, err := src.Sync(context.Background(), nil)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}
	printItems(t, items)
}

// TestJiraSignalStatusChanged verifies that the status_changed signal fires on
// incremental syncs when a ticket's status has changed since the cursor.
//
// This test performs two syncs:
//  1. A full sync from time.Now() — captures the current state, no signal fires
//     (first/full syncs always skip status_changed to avoid noise).
//  2. An incremental sync anchored 30 days ago — any ticket whose status
//     changed in the last 30 days should fire the signal.
//
// Validate: items from the second sync should show "status_changed" in "why".
// Open each ticket's history in Jira to confirm a status transition occurred
// in the last 30 days.
func TestJiraSignalStatusChanged(t *testing.T) {
	token, email, host, jql := integrationEnv(t)

	cfg := baseConfig(token, email, host, jql)
	cfg.WaitsOnMe = []jira.WaitsOnMeSignal{jira.WaitsOnMeStatusChanged}
	src := jira.New("e2e-status-changed", cfg)

	// Full sync — status_changed is always skipped on first sync.
	fullItems, err := src.Sync(context.Background(), nil)
	if err != nil {
		t.Fatalf("full Sync failed: %v", err)
	}
	t.Logf("full sync (status_changed suppressed): %d items", len(fullItems))
	for _, item := range fullItems {
		if item.WaitsOnMe {
			t.Errorf("full sync should not fire status_changed, but item %s has WaitsOnMe=true", item.ID)
		}
	}

	// Incremental sync anchored 30 days ago — any status change in that window fires.
	since := time.Now().Add(-30 * 24 * time.Hour)
	incrItems, err := src.Sync(context.Background(), &since)
	if err != nil {
		t.Fatalf("incremental Sync failed: %v", err)
	}
	t.Logf("incremental sync (30d window): %d items", len(incrItems))
	printItems(t, incrItems)
}

// TestJiraSignalAll runs every signal as a subtest so you can compare their
// outputs side-by-side in one go. Useful for a full audit of your Jira boards.
func TestJiraSignalAll(t *testing.T) {
	token, email, host, jql := integrationEnv(t)

	signals := []jira.WaitsOnMeSignal{
		jira.WaitsOnMeAssigned,
		jira.WaitsOnMeCommentReceived,
		jira.WaitsOnMeStale,
	}

	for _, sig := range signals {
		t.Run(string(sig), func(t *testing.T) {
			cfg := baseConfig(token, email, host, jql)
			cfg.WaitsOnMe = []jira.WaitsOnMeSignal{sig}
			src := jira.New("e2e-signal", cfg)

			items, err := src.Sync(context.Background(), nil)
			if err != nil {
				t.Fatalf("Sync failed: %v", err)
			}
			printItems(t, items)
		})
	}
}
