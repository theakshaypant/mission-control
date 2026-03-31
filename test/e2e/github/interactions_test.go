//go:build integration

package e2e

// TestGitHubPRInteractions shows how UserActivityAt changes depending on which
// interaction types are configured. The set of PRs returned is the same
// across all subtests — interactions only affect the "my activity" timestamp,
// not which PRs are surfaced (that is driven by waits_on_me signals).
//
// This test uses author_updated so it only returns PRs you have previously
// engaged with. Compare the "my activity" line across subtests to see the
// effect of narrowing the interaction type.
//
// Example: with approve_only, a PR where you left a comment (not an approval)
// will still surface via author_updated (raw activity is always checked for
// signals), but "my activity" will show — because comments don't count under
// approve_only.

import (
	"context"
	"testing"

	github "github.com/theakshaypant/mission-control/internal/sources/github"
)

func TestGitHubPRInteractions(t *testing.T) {
	token, user, repos := integrationEnv(t)

	cases := []struct {
		name         string
		interactions []github.Interaction
	}{
		{"all_default", nil},
		{"review_only", []github.Interaction{github.InteractionReview}},
		{"approve_only", []github.Interaction{github.InteractionApprove}},
		{"comment_only", []github.Interaction{github.InteractionComment}},
		{"request_changes_only", []github.Interaction{github.InteractionRequestChanges}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := baseConfig(token, user, repos)
			// Use a single signal that requires prior engagement so the result
			// set is small and the UserActivityAt difference is easy to spot.
			cfg.WaitsOnMe = []github.WaitsOnMeSignal{github.WaitsOnMeAuthorUpdated}
			cfg.Interactions = tc.interactions
			src := github.New("e2e-interactions", cfg)

			items, err := src.Sync(context.Background())
			if err != nil {
				t.Fatalf("Sync failed: %v", err)
			}
			printItems(t, items)
		})
	}
}
