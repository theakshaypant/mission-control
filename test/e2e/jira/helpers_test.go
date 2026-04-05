//go:build integration

// Package e2e contains end-to-end tests for the Jira source. These tests hit
// the real Jira Cloud REST API and are skipped unless the following environment
// variables are set:
//
//	JIRA_TOKEN  — an Atlassian API token
//	JIRA_EMAIL  — the email address associated with the token
//	JIRA_HOST   — bare hostname of the Jira Cloud site, e.g. "mycompany.atlassian.net"
//	JIRA_JQL    — a JQL query that returns tickets relevant to the user, e.g.
//	              "assignee = currentUser() AND statusCategory != Done"
//
// Run all Jira e2e tests:
//
//	JIRA_TOKEN=... JIRA_EMAIL=... JIRA_HOST=... JIRA_JQL="assignee = currentUser()" \
//	  go test -v -tags integration ./test/e2e/jira/
//
// Run a specific test:
//
//	... go test -v -tags integration -run TestJiraSignalAssigned ./test/e2e/jira/
package e2e

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/theakshaypant/mission-control/internal/core"
	"github.com/theakshaypant/mission-control/internal/sources/jira"
)

// integrationEnv reads the required environment variables and skips the test
// if any are missing.
func integrationEnv(t *testing.T) (token, email, host, jql string) {
	t.Helper()
	token = os.Getenv("JIRA_TOKEN")
	email = os.Getenv("JIRA_EMAIL")
	host = os.Getenv("JIRA_HOST")
	jql = os.Getenv("JIRA_JQL")
	if token == "" || email == "" || host == "" || jql == "" {
		t.Skip("set JIRA_TOKEN, JIRA_EMAIL, JIRA_HOST, and JIRA_JQL to run integration tests")
	}
	return token, email, host, jql
}

// baseConfig returns a Config with one board using the provided JQL, all four
// default signals active, and a generous max_results.
func baseConfig(token, email, host, jql string) *jira.Config {
	return &jira.Config{
		Host:  host,
		Email: email,
		Token: token,
		Boards: []jira.Board{
			{Name: "Test Board", JQL: jql, MaxResults: 50},
		},
	}
}

// printItems logs a human-readable summary of every item for manual validation
// against the Jira UI.
func printItems(t *testing.T, items []core.Item) {
	t.Helper()

	activityCount := 0
	for _, item := range items {
		if item.UserActivityAt != nil {
			activityCount++
		}
	}
	t.Logf("─── %d items  (my activity set: %d) ───", len(items), activityCount)

	for _, item := range items {
		if item.Closed {
			t.Logf("")
			t.Logf("  %s  [tombstone]", item.ID)
			continue
		}

		activity := "—"
		if item.UserActivityAt != nil {
			activity = item.UserActivityAt.Format("2006-01-02T15:04:05Z")
		}

		var attrs jira.TicketAttributes
		if len(item.Attributes) > 0 {
			_ = json.Unmarshal(item.Attributes, &attrs)
		}

		t.Logf("")
		t.Logf("  %s  [%s]", item.ID, item.Type)
		t.Logf("    title:          %s", item.Title)
		t.Logf("    url:            %s", item.URL)
		t.Logf("    namespace:      %s", item.Namespace)
		t.Logf("    created:        %s", item.CreatedAt.Format("2006-01-02T15:04:05Z"))
		t.Logf("    updated:        %s", item.UpdatedAt.Format("2006-01-02T15:04:05Z"))
		t.Logf("    my activity:    %s", activity)
		t.Logf("    waits on me:    %v", item.WaitsOnMe)
		t.Logf("    assigned to me: %v", item.IsAssigned)
		t.Logf("    status:         %s", attrs.Status)
		t.Logf("    issue type:     %s", attrs.IssueType)
		if attrs.Priority != "" {
			t.Logf("    priority:       %s", attrs.Priority)
		}
		if attrs.Assignee != "" {
			t.Logf("    assignee:       %s", attrs.Assignee)
		}
		if attrs.Reporter != "" {
			t.Logf("    reporter:       %s", attrs.Reporter)
		}
		if len(attrs.ActiveSignals) > 0 {
			t.Logf("    why:            %s", strings.Join(attrs.ActiveSignals, ", "))
		}
		if len(attrs.Labels) > 0 {
			t.Logf("    labels:         %s", strings.Join(attrs.Labels, ", "))
		}
	}
}
