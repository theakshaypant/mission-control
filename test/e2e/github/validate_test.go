//go:build integration

package e2e

import (
	"strings"
	"testing"

	github "github.com/theakshaypant/mission-control/internal/sources/github"
)

// TestConfig_Validate confirms that well-formed configs pass validation and
// broken ones fail with the expected error substring. No API calls are made —
// dummy credentials are used so this test runs without real env vars.
func TestGitHubConfigValidate(t *testing.T) {
	// Use fixed dummy values — no API calls are made in this test.
	const (
		tok  = "ghp_test"
		user = "octocat"
	)
	repos := []string{"owner/repo"}

	cases := []struct {
		name    string
		cfg     github.Config
		wantErr string // empty = expect no error
	}{
		{
			name: "valid config",
			cfg:  github.Config{Token: tok, User: user, Repos: repos},
		},
		{
			name:    "missing token",
			cfg:     github.Config{User: user, Repos: repos},
			wantErr: "token is required",
		},
		{
			name:    "missing user",
			cfg:     github.Config{Token: tok, Repos: repos},
			wantErr: "user is required",
		},
		{
			name:    "missing repos",
			cfg:     github.Config{Token: tok, User: user},
			wantErr: "at least one repo is required",
		},
		{
			name:    "bad repo format",
			cfg:     github.Config{Token: tok, User: user, Repos: []string{"noslash"}},
			wantErr: "owner/repo format",
		},
		{
			name:    "bad pr_scope",
			cfg:     github.Config{Token: tok, User: user, Repos: repos, PRScope: "bogus"},
			wantErr: "unknown pr_scope",
		},
		{
			name: "pr_scope none",
			cfg:  github.Config{Token: tok, User: user, Repos: repos, PRScope: github.FetchScopeNone},
		},
		{
			name:    "bad issue_scope",
			cfg:     github.Config{Token: tok, User: user, Repos: repos, IssueScope: "bogus"},
			wantErr: "unknown issue_scope",
		},
		{
			name: "issue_scope involved",
			cfg:  github.Config{Token: tok, User: user, Repos: repos, IssueScope: github.FetchScopeInvolved},
		},
		{
			name: "issue_scope all",
			cfg:  github.Config{Token: tok, User: user, Repos: repos, IssueScope: github.FetchScopeAll},
		},
		{
			name:    "negative issue_updated_within_days",
			cfg:     github.Config{Token: tok, User: user, Repos: repos, IssueUpdatedWithinDays: -1},
			wantErr: "issue_updated_within_days must be non-negative",
		},
		{
			name: "issue_updated_within_days zero (disabled)",
			cfg:  github.Config{Token: tok, User: user, Repos: repos, IssueUpdatedWithinDays: 0},
		},
		{
			name:    "issue_comment_limit too high",
			cfg:     github.Config{Token: tok, User: user, Repos: repos, IssueCommentLimit: 101},
			wantErr: "issue_comment_limit must be between",
		},
		{
			name:    "issue_comment_limit negative",
			cfg:     github.Config{Token: tok, User: user, Repos: repos, IssueCommentLimit: -1},
			wantErr: "issue_comment_limit must be between",
		},
		{
			name: "issue_comment_limit at max",
			cfg:  github.Config{Token: tok, User: user, Repos: repos, IssueCommentLimit: 100},
		},
		{
			name:    "bad interaction",
			cfg:     github.Config{Token: tok, User: user, Repos: repos, Interactions: []github.Interaction{"bogus"}},
			wantErr: "unknown interaction type",
		},
		{
			name:    "bad waits_on_me signal",
			cfg:     github.Config{Token: tok, User: user, Repos: repos, WaitsOnMe: []github.WaitsOnMeSignal{"bogus"}},
			wantErr: "unknown waits_on_me signal",
		},
		{
			name: "valid is_assigned: reviewer",
			cfg:  github.Config{Token: tok, User: user, Repos: repos, IsAssigned: []github.AssignedSignal{github.AssignedSignalReviewer}},
		},
		{
			name:    "bad is_assigned value",
			cfg:     github.Config{Token: tok, User: user, Repos: repos, IsAssigned: []github.AssignedSignal{"bogus"}},
			wantErr: "unknown is_assigned signal",
		},
		{
			name:    "negative stale_days",
			cfg:     github.Config{Token: tok, User: user, Repos: repos, StaleDays: -1},
			wantErr: "stale_days must be non-negative",
		},
		{
			name: "valid GHE host",
			cfg:  github.Config{Token: tok, User: user, Repos: repos, Host: "github.mycompany.com"},
		},
		{
			name:    "host with protocol",
			cfg:     github.Config{Token: tok, User: user, Repos: repos, Host: "https://github.mycompany.com"},
			wantErr: "bare hostname",
		},
		{
			name:    "host with path",
			cfg:     github.Config{Token: tok, User: user, Repos: repos, Host: "github.mycompany.com/api"},
			wantErr: "bare hostname",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			if tc.wantErr == "" {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("expected error containing %q, got: %v", tc.wantErr, err)
			}
		})
	}
}
