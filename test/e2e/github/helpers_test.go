//go:build integration

// Package e2e contains end-to-end tests for the GitHub source. These tests hit
// the real GitHub API and are skipped unless the following environment variables
// are set:
//
//	GITHUB_TOKEN      — a personal access token with repo read access
//	GITHUB_USER       — the GitHub login associated with the token
//	GITHUB_TEST_REPOS — comma-separated list of repos in "owner/repo" format
//
// Run all GitHub e2e tests:
//
//	GITHUB_TOKEN=... GITHUB_USER=... GITHUB_TEST_REPOS=owner/repo \
//	  go test -v -tags integration ./test/e2e/github/
//
// Run a specific test:
//
//	... go test -v -tags integration -run TestSignals_Unreviewed ./test/e2e/github/
package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/theakshaypant/mission-control/internal/core"
	github "github.com/theakshaypant/mission-control/internal/sources/github"
)

// integrationEnv reads the required environment variables and skips the test
// if any are missing.
func integrationEnv(t *testing.T) (token, user string, repos []string) {
	t.Helper()
	token = os.Getenv("GITHUB_TOKEN")
	user = os.Getenv("GITHUB_USER")
	reposRaw := os.Getenv("GITHUB_TEST_REPOS")
	if token == "" || user == "" || reposRaw == "" {
		t.Skip("set GITHUB_TOKEN, GITHUB_USER, and GITHUB_TEST_REPOS to run integration tests")
	}
	for r := range strings.SplitSeq(reposRaw, ",") {
		if r = strings.TrimSpace(r); r != "" {
			repos = append(repos, r)
		}
	}
	return token, user, repos
}

// baseConfig returns a Config with the maintainer defaults: fetch all open PRs,
// no artificial cap, default signals.
func baseConfig(token, user string, repos []string) *github.Config {
	return &github.Config{
		Token:   token,
		User:    user,
		Repos:   repos,
		PRScope: github.FetchScopeAll,
		MaxPRs:  100,
	}
}

// baseIssueConfig returns a Config with issue syncing enabled via the involved
// scope and PRs explicitly disabled, suitable for issue-specific tests.
func baseIssueConfig(token, user string, repos []string) *github.Config {
	return &github.Config{
		Token:      token,
		User:       user,
		Repos:      repos,
		PRScope:    github.FetchScopeNone,
		IssueScope: github.FetchScopeInvolved,
		MaxIssues:  100,
	}
}

// printItems logs a human-readable summary of every item for manual validation
// against the GitHub UI.
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
		activity := "—"
		if item.UserActivityAt != nil {
			activity = item.UserActivityAt.Format("2006-01-02T15:04:05Z")
		}

		t.Logf("")
		t.Logf("  %s  [%s]", item.ID, item.Type)
		t.Logf("    title:          %s", item.Title)
		t.Logf("    url:            %s", item.URL)
		t.Logf("    created:        %s", item.CreatedAt.Format("2006-01-02T15:04:05Z"))
		t.Logf("    updated:        %s", item.UpdatedAt.Format("2006-01-02T15:04:05Z"))
		t.Logf("    my activity:    %s", activity)
		t.Logf("    assigned to me: %v", item.IsAssigned)

		switch item.Type {
		case github.TypePR:
			var attrs github.PRAttributes
			if len(item.Attributes) > 0 {
				_ = json.Unmarshal(item.Attributes, &attrs)
			}
			t.Logf("    author:         %s", attrs.Author)
			t.Logf("    review state:   %s", fmtReviewDecision(attrs.ReviewDecision))
			t.Logf("    draft:          %v", attrs.IsDraft)
			if len(attrs.ActiveSignals) > 0 {
				t.Logf("    why:            %s", strings.Join(attrs.ActiveSignals, ", "))
			}
			if len(attrs.Labels) > 0 {
				t.Logf("    labels:         %s", strings.Join(attrs.Labels, ", "))
			}
		case github.TypeIssue:
			var attrs github.IssueAttributes
			if len(item.Attributes) > 0 {
				_ = json.Unmarshal(item.Attributes, &attrs)
			}
			t.Logf("    author:         %s", attrs.Author)
			if len(attrs.ActiveSignals) > 0 {
				t.Logf("    why:            %s", strings.Join(attrs.ActiveSignals, ", "))
			}
			if len(attrs.Labels) > 0 {
				t.Logf("    labels:         %s", strings.Join(attrs.Labels, ", "))
			}
		}
	}
}

func fmtReviewDecision(d string) string {
	switch d {
	case "APPROVED":
		return "approved"
	case "CHANGES_REQUESTED":
		return "changes requested"
	case "REVIEW_REQUIRED":
		return "review required"
	case "":
		return "—"
	default:
		return fmt.Sprintf("unknown (%s)", d)
	}
}
