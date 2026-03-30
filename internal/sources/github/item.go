package github

import (
	"fmt"

	"github.com/theakshaypant/mission-control/internal/core"
)

const (
	Kind      core.SourceKind = "github"
	TypePR    core.ItemType   = "pr"
	TypeIssue core.ItemType   = "issue"
)

// PRAttributes holds GitHub-specific data for a pull request Item.
// Mapped onto Item.Attributes as JSON.
type PRAttributes struct {
	// login of the PR author
	Author   string   `json:"author"`
	Labels   []string `json:"labels,omitempty"`
	IsDraft  bool     `json:"is_draft,omitempty"`
	IsMerged bool     `json:"is_merged,omitempty"`
	// "open", "closed"
	State string `json:"state"`
	// APPROVED, CHANGES_REQUESTED, REVIEW_REQUIRED
	ReviewDecision string `json:"review_decision,omitempty"`
	// waits_on_me signals that fired
	ActiveSignals []string `json:"active_signals,omitempty"`
}

// IssueAttributes holds GitHub-specific data for an issue Item.
// Mapped onto Item.Attributes as JSON.
type IssueAttributes struct {
	// login of the issue author
	Author string   `json:"author"`
	Labels []string `json:"labels,omitempty"`
	// "open"
	State string `json:"state"`
	// waits_on_me signals that fired
	ActiveSignals []string `json:"active_signals,omitempty"`
}

// ItemID produces the stable ID for a GitHub PR or issue.
func ItemID(repo string, number int) string {
	return fmt.Sprintf("github:%s#%d", repo, number)
}
