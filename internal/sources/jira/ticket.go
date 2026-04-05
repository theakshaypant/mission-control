package jira

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/theakshaypant/mission-control/internal/core"
)

const (
	defaultStaleDays  = 14
	defaultMaxResults = 50
)

// normalizeTicket maps a raw Jira issue onto a core.Item.
// boardName is used as Item.Namespace. since is the incremental sync cursor,
// used to scope the status_changed signal; nil means a full (first) sync.
func (s *Source) normalizeTicket(issue issueNode, boardName string, since *time.Time) core.Item {
	if s.isDone(issue.Fields.Status.Name) {
		return core.Item{
			ID:     ItemID(s.name, issue.Key),
			Closed: true,
		}
	}

	cfg := s.config
	user := cfg.Email

	comments := issue.Fields.Comment.Comments
	userLastComment := latestCommentBy(comments, user)
	latestOtherComment := latestCommentExcluding(comments, user)

	staleDays := cfg.StaleDays
	if staleDays == 0 {
		staleDays = defaultStaleDays
	}

	waitsSignals := cfg.WaitsOnMe
	if len(waitsSignals) == 0 {
		waitsSignals = []WaitsOnMeSignal{
			WaitsOnMeAssigned,
			WaitsOnMeCommentReceived,
			WaitsOnMeStale,
			WaitsOnMeStatusChanged,
		}
	}

	waitsOnMe := false
	var activeSignals []string
	for _, sig := range waitsSignals {
		fired := false
		switch sig {
		case WaitsOnMeAssigned:
			if issue.Fields.Assignee != nil && issue.Fields.Assignee.EmailAddress == user {
				fired = true
			}
		case WaitsOnMeCommentReceived:
			// Fire if someone else has commented more recently than the user's last
			// comment, and the user is the assignee or reporter of the ticket.
			isInvolved := (issue.Fields.Assignee != nil && issue.Fields.Assignee.EmailAddress == user) ||
				(issue.Fields.Reporter != nil && issue.Fields.Reporter.EmailAddress == user)
			if isInvolved && latestOtherComment != nil {
				if userLastComment == nil || latestOtherComment.After(*userLastComment) {
					fired = true
				}
			}
		case WaitsOnMeStale:
			if time.Since(issue.Fields.Updated.Time) >= time.Duration(staleDays)*24*time.Hour {
				fired = true
			}
		case WaitsOnMeStatusChanged:
			// Only fire on incremental syncs. On a full (first) sync every ticket
			// has prior status changes, which would create noise.
			if since != nil {
				for _, h := range issue.Changelog.Histories {
					if h.Created.After(*since) {
						for _, item := range h.Items {
							if item.Field == "status" {
								fired = true
								break
							}
						}
					}
					if fired {
						break
					}
				}
			}
		}
		if fired {
			waitsOnMe = true
			activeSignals = append(activeSignals, string(sig))
		}
	}

	// UserActivityAt: most recent comment posted by the user.
	// Only comment interactions are supported in v1.
	var userActivityAt *time.Time
	if userLastComment != nil {
		t := *userLastComment
		userActivityAt = &t
	}

	isAssigned := issue.Fields.Assignee != nil && issue.Fields.Assignee.EmailAddress == user

	attrs := TicketAttributes{
		IssueKey:      issue.Key,
		IssueType:     issue.Fields.IssueType.Name,
		Status:        issue.Fields.Status.Name,
		Labels:        issue.Fields.Labels,
		ActiveSignals: activeSignals,
	}
	if issue.Fields.Priority != nil {
		attrs.Priority = issue.Fields.Priority.Name
	}
	if issue.Fields.Reporter != nil {
		attrs.Reporter = issue.Fields.Reporter.EmailAddress
	}
	if issue.Fields.Assignee != nil {
		attrs.Assignee = issue.Fields.Assignee.EmailAddress
	}
	attrsJSON, _ := json.Marshal(attrs)

	return core.Item{
		ID:             ItemID(s.name, issue.Key),
		Source:         Kind,
		Type:           itemTypeFor(issue.Fields.IssueType.Name),
		Title:          issue.Fields.Summary,
		URL:            fmt.Sprintf("https://%s/browse/%s", s.config.Host, issue.Key),
		Namespace:      boardName,
		CreatedAt:      issue.Fields.Created.Time,
		UpdatedAt:      issue.Fields.Updated.Time,
		WaitsOnMe:      waitsOnMe,
		IsAssigned:     isAssigned,
		UserActivityAt: userActivityAt,
		Attributes:     attrsJSON,
	}
}

// mergeItems merges two items for the same issue key (produced by different boards).
// The first item's namespace is preserved. Active signals are unioned.
// A tombstone (Closed: true) from either side takes precedence.
func mergeItems(existing, incoming core.Item) core.Item {
	if incoming.Closed {
		return incoming
	}
	if existing.Closed {
		return existing
	}
	if incoming.WaitsOnMe {
		existing.WaitsOnMe = true
	}
	if incoming.IsAssigned {
		existing.IsAssigned = true
	}
	if incoming.UserActivityAt != nil {
		if existing.UserActivityAt == nil || incoming.UserActivityAt.After(*existing.UserActivityAt) {
			existing.UserActivityAt = incoming.UserActivityAt
		}
	}

	// Union active signals from both items' Attributes.
	var existingAttrs, incomingAttrs TicketAttributes
	_ = json.Unmarshal(existing.Attributes, &existingAttrs)
	_ = json.Unmarshal(incoming.Attributes, &incomingAttrs)

	sigSet := make(map[string]struct{}, len(existingAttrs.ActiveSignals)+len(incomingAttrs.ActiveSignals))
	for _, s := range existingAttrs.ActiveSignals {
		sigSet[s] = struct{}{}
	}
	for _, s := range incomingAttrs.ActiveSignals {
		sigSet[s] = struct{}{}
	}
	merged := make([]string, 0, len(sigSet))
	for s := range sigSet {
		merged = append(merged, s)
	}
	existingAttrs.ActiveSignals = merged
	existing.Attributes, _ = json.Marshal(existingAttrs)
	return existing
}

// isDone reports whether the given status name is in the configured done list.
// Comparison is case-insensitive.
func (s *Source) isDone(status string) bool {
	for _, d := range s.doneStatuses() {
		if strings.EqualFold(status, d) {
			return true
		}
	}
	return false
}

// doneStatuses returns the configured terminal statuses, or a sensible default.
func (s *Source) doneStatuses() []string {
	if len(s.config.DoneStatuses) > 0 {
		return s.config.DoneStatuses
	}
	return []string{"Done", "Closed", "Resolved", "Won't Do"}
}

// latestCommentBy returns the timestamp of the most recent comment whose
// author email matches login, or nil if none exists.
func latestCommentBy(comments []commentNode, email string) *time.Time {
	var latest *time.Time
	for _, c := range comments {
		if c.Author.EmailAddress == email {
			t := c.Created.Time
			if latest == nil || t.After(*latest) {
				latest = &t
			}
		}
	}
	return latest
}

// latestCommentExcluding returns the timestamp of the most recent comment
// whose author email does not match the excluded email, or nil if none exists.
func latestCommentExcluding(comments []commentNode, excludeEmail string) *time.Time {
	var latest *time.Time
	for _, c := range comments {
		if c.Author.EmailAddress != excludeEmail {
			t := c.Created.Time
			if latest == nil || t.After(*latest) {
				latest = &t
			}
		}
	}
	return latest
}

// addUpdatedFilter appends an `updated > "{since}"` clause to a JQL string,
// injecting it before any ORDER BY clause so the query remains valid.
func addUpdatedFilter(jql string, since time.Time) string {
	sinceStr := fmt.Sprintf(` AND updated > "%s"`, since.UTC().Format("2006-01-02 15:04"))
	// Find the last occurrence of ORDER BY (case-insensitive) and inject before it.
	upper := strings.ToUpper(jql)
	if idx := strings.LastIndex(upper, " ORDER BY "); idx >= 0 {
		return jql[:idx] + sinceStr + jql[idx:]
	}
	return jql + sinceStr
}
