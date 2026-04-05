package jira

import (
	"fmt"
	"strings"

	"github.com/theakshaypant/mission-control/internal/core"
)

const (
	Kind core.SourceKind = "jira"

	// ItemType constants derived from the Jira issue type field.
	TypeBug     core.ItemType = "bug"
	TypeStory   core.ItemType = "story"
	TypeTask    core.ItemType = "task"
	TypeEpic    core.ItemType = "epic"
	TypeFeature core.ItemType = "feature"
	// TypeTicket is the fallback for any issue type not matched above.
	TypeTicket core.ItemType = "ticket"
)

// itemTypeFor maps a Jira issue type name (case-insensitive) to a core.ItemType.
// Unknown types fall back to TypeTicket.
func itemTypeFor(jiraTypeName string) core.ItemType {
	switch strings.ToLower(jiraTypeName) {
	case "bug":
		return TypeBug
	case "story":
		return TypeStory
	case "task", "sub-task", "subtask":
		return TypeTask
	case "epic":
		return TypeEpic
	case "feature":
		return TypeFeature
	default:
		return TypeTicket
	}
}

// TicketAttributes holds Jira-specific data for a ticket Item.
// Mapped onto Item.Attributes as JSON.
type TicketAttributes struct {
	// IssueKey is the Jira issue key, e.g. "PROJ-123".
	IssueKey string `json:"issue_key"`
	// IssueType is the Jira issue type name, e.g. "Bug", "Story", "Task".
	IssueType string `json:"issue_type"`
	// Status is the current workflow status name, e.g. "In Review".
	Status string `json:"status"`
	// Priority is the issue priority name, e.g. "High". Empty if unset.
	Priority string `json:"priority,omitempty"`
	// Reporter is the emailAddress of the issue reporter. Empty if unset.
	Reporter string `json:"reporter,omitempty"`
	// Assignee is the emailAddress of the current assignee. Empty if unset.
	Assignee string `json:"assignee,omitempty"`
	// Labels holds any labels attached to the issue.
	Labels []string `json:"labels,omitempty"`
	// ActiveSignals lists the waits_on_me signals that fired for this item.
	ActiveSignals []string `json:"active_signals,omitempty"`
}

// ItemID produces the stable ID for a Jira ticket within a named source instance.
// sourceName is the user-defined instance name (e.g. "work"), not the kind.
func ItemID(sourceName, issueKey string) string {
	return fmt.Sprintf("jira:%s:%s", sourceName, issueKey)
}
