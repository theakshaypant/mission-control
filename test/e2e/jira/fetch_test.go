//go:build integration

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/theakshaypant/mission-control/internal/sources/jira"
)

// TestJiraBoardFetch confirms that a basic board fetch returns items and that
// each item has the expected fields populated.
//
// Validate: check that titles, URLs, and statuses look correct against Jira.
func TestJiraBoardFetch(t *testing.T) {
	token, email, host, jql := integrationEnv(t)

	src := jira.New("e2e-fetch", baseConfig(token, email, host, jql))

	items, err := src.Sync(context.Background(), nil)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	for _, item := range items {
		if item.Closed {
			continue
		}
		if item.ID == "" {
			t.Errorf("item has empty ID: %+v", item)
		}
		if item.Title == "" {
			t.Errorf("item %s has empty Title", item.ID)
		}
		if item.URL == "" {
			t.Errorf("item %s has empty URL", item.ID)
		}
		if item.CreatedAt.IsZero() {
			t.Errorf("item %s has zero CreatedAt", item.ID)
		}
		if item.UpdatedAt.IsZero() {
			t.Errorf("item %s has zero UpdatedAt", item.ID)
		}
		if item.Namespace == "" {
			t.Errorf("item %s has empty Namespace", item.ID)
		}
	}

	printItems(t, items)
}

// TestJiraMaxResults confirms that max_results is respected as a per-board cap.
//
// Validate: the item count should not exceed the configured limit.
func TestJiraMaxResults(t *testing.T) {
	token, email, host, jql := integrationEnv(t)

	const limit = 5
	cfg := baseConfig(token, email, host, jql)
	cfg.Boards[0].MaxResults = limit
	src := jira.New("e2e-max-results", cfg)

	items, err := src.Sync(context.Background(), nil)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// Exclude tombstones from the count — they come from the done-status query.
	open := 0
	for _, item := range items {
		if !item.Closed {
			open++
		}
	}
	if open > limit {
		t.Errorf("got %d open items, expected at most %d", open, limit)
	}
	t.Logf("got %d open items (limit %d)", open, limit)
	printItems(t, items)
}

// TestJiraMultipleBoards confirms that a source with two boards returns
// deduplicated results — a ticket matching both boards appears only once.
//
// The second board uses the same JQL as the first; every ticket will match
// both. After dedup the count should equal a single-board fetch.
//
// Validate: item count from two identical boards should match one-board count.
func TestJiraMultipleBoards(t *testing.T) {
	token, email, host, jql := integrationEnv(t)

	singleCfg := baseConfig(token, email, host, jql)
	singleSrc := jira.New("e2e-single-board", singleCfg)
	singleItems, err := singleSrc.Sync(context.Background(), nil)
	if err != nil {
		t.Fatalf("single-board Sync failed: %v", err)
	}

	doubleCfg := &jira.Config{
		Host:  host,
		Email: email,
		Token: token,
		Boards: []jira.Board{
			{Name: "Board A", JQL: jql, MaxResults: 50},
			{Name: "Board B", JQL: jql, MaxResults: 50},
		},
	}
	doubleSrc := jira.New("e2e-double-board", doubleCfg)
	doubleItems, err := doubleSrc.Sync(context.Background(), nil)
	if err != nil {
		t.Fatalf("double-board Sync failed: %v", err)
	}

	t.Logf("single board: %d items", len(singleItems))
	t.Logf("double board: %d items (expected same — dedup should collapse duplicates)", len(doubleItems))

	if len(doubleItems) != len(singleItems) {
		t.Errorf("expected %d items after dedup, got %d", len(singleItems), len(doubleItems))
	}

	// Confirm namespace is taken from the first board ("Board A").
	for _, item := range doubleItems {
		if !item.Closed && item.Namespace != "Board A" {
			t.Errorf("item %s: expected namespace %q (first board wins), got %q", item.ID, "Board A", item.Namespace)
		}
	}
}

// TestJiraIssueTypes confirms that the item type field reflects the Jira issue
// type rather than always being "ticket".
//
// Validate: items should have type values like "bug", "story", "task", "epic",
// "feature", or "ticket" (fallback). Check a sample against the Jira UI.
func TestJiraIssueTypes(t *testing.T) {
	token, email, host, jql := integrationEnv(t)

	src := jira.New("e2e-types", baseConfig(token, email, host, jql))

	items, err := src.Sync(context.Background(), nil)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	typeCounts := make(map[string]int)
	for _, item := range items {
		if !item.Closed {
			typeCounts[string(item.Type)]++
		}
	}
	t.Logf("type distribution:")
	for typ, count := range typeCounts {
		t.Logf("  %-10s %d", typ, count)
	}
	printItems(t, items)
}

// TestJiraIncrementalSync verifies that a second sync immediately after the
// first returns no open items — all tickets predate the since filter.
//
// Validate: second sync should log "0 items". Any items returned are
// tombstones from the done-status query (expected to be 0 as well if nothing
// was resolved in the last second).
func TestJiraIncrementalSync(t *testing.T) {
	token, email, host, jql := integrationEnv(t)

	src := jira.New("e2e-incremental", baseConfig(token, email, host, jql))

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
	t.Logf("second sync: %d items (expected 0 — all tickets predate since)", len(second))

	for _, item := range second {
		if !item.Closed {
			t.Errorf("unexpected open item on incremental sync: %s  updated=%s",
				item.ID, item.UpdatedAt.Format("2006-01-02T15:04:05Z"))
		}
	}
}
