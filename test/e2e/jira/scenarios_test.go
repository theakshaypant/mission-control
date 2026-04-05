//go:build integration

package e2e

// Real-world scenario tests. These mirror example configurations from the docs
// and are the closest thing to a production end-to-end test.

import (
	"context"
	"testing"

	"github.com/theakshaypant/mission-control/internal/sources/jira"
)

// TestJiraScenarioMyTickets runs the recommended config for someone who wants
// to track their own assigned work: assigned + comment_received signals only.
//
// Validate: every item should be assigned to you or have a comment from someone
// else that you haven't replied to. Check a few against the Jira UI.
func TestJiraScenarioMyTickets(t *testing.T) {
	token, email, host, jql := integrationEnv(t)

	cfg := baseConfig(token, email, host, jql)
	cfg.WaitsOnMe = []jira.WaitsOnMeSignal{
		jira.WaitsOnMeAssigned,
		jira.WaitsOnMeCommentReceived,
	}
	src := jira.New("e2e-my-tickets", cfg)

	items, err := src.Sync(context.Background(), nil)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}
	printItems(t, items)
}

// TestJiraScenarioStaleBoard simulates the "Stale In Review" board pattern:
// a board JQL that already filters to stale tickets (updated <= -14d). The
// stale signal fires redundantly here — both the JQL and the signal agree.
// This confirms that signal evaluation and JQL scoping compose correctly.
//
// Validate: every item should have an updated date more than 14 days ago.
func TestJiraScenarioStaleBoard(t *testing.T) {
	token, email, host, _ := integrationEnv(t)

	// Override JQL with a stale-focused query. If this returns no results your
	// Jira instance may have no tickets updated more than 14 days ago — that's OK.
	staleJQL := "statusCategory != Done AND updated <= -14d ORDER BY updated ASC"

	cfg := &jira.Config{
		Host:  host,
		Email: email,
		Token: token,
		Boards: []jira.Board{
			{Name: "Stale In Review", JQL: staleJQL, MaxResults: 30},
		},
		WaitsOnMe: []jira.WaitsOnMeSignal{jira.WaitsOnMeStale},
		StaleDays: 14,
	}
	src := jira.New("e2e-stale-board", cfg)

	items, err := src.Sync(context.Background(), nil)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}
	t.Logf("stale tickets (>14d no activity): %d", len(items))
	printItems(t, items)
}

// TestJiraScenarioAllSignals runs all four signals together — the default
// production configuration. This is the scenario a typical Jira user would run.
//
// Validate: check that "why" field makes sense for each item. Assigned tickets
// should show "assigned". Tickets with unread comments should show
// "comment_received". Items with no recent activity should show "stale".
func TestJiraScenarioAllSignals(t *testing.T) {
	token, email, host, jql := integrationEnv(t)

	// Use the default config — all four signals, stale_days 14.
	src := jira.New("e2e-all-signals", baseConfig(token, email, host, jql))

	items, err := src.Sync(context.Background(), nil)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}
	printItems(t, items)
}

// TestJiraScenarioCustomDoneStatuses confirms that tickets in a custom done
// status are tombstoned (Closed: true) rather than upserted.
//
// The test sets done_statuses to include a status that likely has tickets in
// your Jira instance. Any ticket in that status that the JQL returns will be
// returned with Closed: true.
//
// Edit doneStatuses to match your workflow's terminal statuses.
func TestJiraScenarioCustomDoneStatuses(t *testing.T) {
	token, email, host, jql := integrationEnv(t)

	// Customise these to match statuses in your Jira instance.
	doneStatuses := []string{"Done", "Closed", "Resolved", "Won't Do", "Cancelled"}

	cfg := baseConfig(token, email, host, jql)
	cfg.DoneStatuses = doneStatuses
	src := jira.New("e2e-done-statuses", cfg)

	items, err := src.Sync(context.Background(), nil)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	tombstones := 0
	for _, item := range items {
		if item.Closed {
			tombstones++
			t.Logf("tombstone: %s", item.ID)
		}
	}
	t.Logf("%d tombstones out of %d total items", tombstones, len(items))
	printItems(t, items)
}
