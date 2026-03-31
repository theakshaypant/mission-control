package github

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/theakshaypant/mission-control/internal/core"
)

// issueFields contains all GraphQL fields fetched for an issue, shared
// between the per-repo query and the search query.
type issueFields struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Author    struct {
		Login string `json:"login"`
	} `json:"author"`
	Assignees struct {
		Nodes []struct {
			Login string `json:"login"`
		} `json:"nodes"`
	} `json:"assignees"`
	Labels struct {
		Nodes []struct {
			Name string `json:"name"`
		} `json:"nodes"`
	} `json:"labels"`
	State    string `json:"state"` // "OPEN" or "CLOSED"
	Comments struct {
		Nodes []struct {
			Author    struct{ Login string `json:"login"` } `json:"author"`
			CreatedAt time.Time                             `json:"createdAt"`
		} `json:"nodes"`
	} `json:"comments"`
}

func (issue issueFields) commentEntries() []commentEntry {
	result := make([]commentEntry, 0, len(issue.Comments.Nodes))
	for _, c := range issue.Comments.Nodes {
		result = append(result, commentEntry{Login: c.Author.Login, CreatedAt: c.CreatedAt})
	}
	return result
}

// issuesByRepoQuery fetches all open issues in a single repo, ordered by most
// recently updated. Supports cursor-based pagination.
const issuesByRepoQuery = `
query IssuesByRepo($owner: String!, $repo: String!, $first: Int!, $after: String, $commentLimit: Int!) {
  repository(owner: $owner, name: $repo) {
    issues(states: [OPEN], first: $first, after: $after, orderBy: {field: UPDATED_AT, direction: DESC}) {
      pageInfo { hasNextPage endCursor }
      nodes {
        number title url createdAt updatedAt state
        author { login }
        assignees(first: 10) { nodes { login } }
        labels(first: 20) { nodes { name } }
        comments(last: $commentLimit) {
          nodes { author { login } createdAt }
        }
      }
    }
  }
}`

type issuesByRepoResponse struct {
	Repository struct {
		Issues struct {
			PageInfo struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
			Nodes []issueFields `json:"nodes"`
		} `json:"issues"`
	} `json:"repository"`
}

// involvedIssuesQuery uses GitHub's search API to fetch open issues across all
// configured repos where the user is involved (author, assignee, or mentioned).
const involvedIssuesQuery = `
query InvolvedIssues($q: String!, $first: Int!, $after: String, $commentLimit: Int!) {
  search(query: $q, type: ISSUE, first: $first, after: $after) {
    pageInfo { hasNextPage endCursor }
    nodes {
      ... on Issue {
        number title url createdAt updatedAt state
        repository { nameWithOwner }
        author { login }
        assignees(first: 10) { nodes { login } }
        labels(first: 20) { nodes { name } }
        comments(last: $commentLimit) {
          nodes { author { login } createdAt }
        }
      }
    }
  }
}`

type involvedIssuesResponse struct {
	Search struct {
		PageInfo struct {
			HasNextPage bool   `json:"hasNextPage"`
			EndCursor   string `json:"endCursor"`
		} `json:"pageInfo"`
		Nodes []struct {
			issueFields
			Repository struct {
				NameWithOwner string `json:"nameWithOwner"`
			} `json:"repository"`
		} `json:"nodes"`
	} `json:"search"`
}

const (
	defaultIssueUpdatedWithinDays = 7
	defaultIssueCommentLimit      = 10
)

func (s *Source) syncIssues(ctx context.Context, sincePtr *time.Time) ([]core.Item, error) {
	scope := s.config.IssueScope
	maxIssues := s.config.MaxIssues
	if maxIssues == 0 {
		maxIssues = defaultMaxPRs
	}

	// since is the incremental sync cursor from the last run.
	var since time.Time
	if sincePtr != nil {
		since = *sincePtr
	}

	// cutoff is the effective lower bound for issue updatedAt. It is the more
	// recent of the incremental sync cursor and the updated_within threshold,
	// so we never fetch issues that are both stale and already seen.
	updatedWithin := s.config.IssueUpdatedWithinDays
	if updatedWithin == 0 {
		updatedWithin = defaultIssueUpdatedWithinDays
	}
	cutoff := since
	if updatedWithin > 0 {
		threshold := time.Now().AddDate(0, 0, -updatedWithin)
		if threshold.After(cutoff) {
			cutoff = threshold
		}
	}

	commentLimit := s.config.IssueCommentLimit
	if commentLimit == 0 {
		commentLimit = defaultIssueCommentLimit
	}

	var items []core.Item
	var err error
	switch scope {
	case FetchScopeAll:
		items, err = s.fetchAllIssues(ctx, cutoff, maxIssues, commentLimit)
	case FetchScopeInvolved:
		items, err = s.fetchInvolvedIssues(ctx, cutoff, maxIssues, commentLimit)
	default:
		return nil, fmt.Errorf("github: unknown issue_scope %q", scope)
	}
	if err != nil {
		return nil, err
	}

	return items, nil
}

// fetchAllIssues retrieves all open issues from each configured repo. Results
// are ordered by most recently updated; pagination stops once all remaining
// issues predate the since time or maxIssues is reached per repo.
func (s *Source) fetchAllIssues(ctx context.Context, since time.Time, maxIssues, commentLimit int) ([]core.Item, error) {
	var allItems []core.Item

	for _, repoStr := range s.config.Repos {
		parts := strings.SplitN(repoStr, "/", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("github: invalid repo format %q", repoStr)
		}
		owner, repo := parts[0], parts[1]

		remaining := maxIssues
		var cursor *string
		for remaining > 0 {
			pageSize := min(remaining, maxPerPage)
			vars := map[string]any{
				"owner":        owner,
				"repo":         repo,
				"first":        pageSize,
				"after":        cursor, // nil on first page
				"commentLimit": commentLimit,
			}

			data, err := doGraphQL[issuesByRepoResponse](ctx, s.config.Token, s.graphqlEndpoint(), issuesByRepoQuery, vars)
			if err != nil {
				return nil, fmt.Errorf("fetch issues for %s: %w", repoStr, err)
			}

			issues := data.Repository.Issues
			done := false
			for _, issue := range issues.Nodes {
				// Since results are ordered DESC by updatedAt, once we see an
				// issue older than our last sync we can stop fetching this repo.
				if !since.IsZero() && !issue.UpdatedAt.After(since) {
					done = true
					break
				}
				allItems = append(allItems, s.normalizeIssue(issue, repoStr))
				remaining--
				if remaining == 0 {
					done = true
					break
				}
			}

			if done || !issues.PageInfo.HasNextPage {
				break
			}
			c := issues.PageInfo.EndCursor
			cursor = &c
		}
	}

	return allItems, nil
}

// fetchInvolvedIssues uses GitHub's search API to fetch open issues across all
// configured repos where the user is involved. A since date filter is added
// when available to limit results to recently updated issues.
func (s *Source) fetchInvolvedIssues(ctx context.Context, since time.Time, maxIssues, commentLimit int) ([]core.Item, error) {
	var sb strings.Builder
	// On the first run (no cursor) restrict to open issues only to avoid
	// pulling in historical closed issues. On incremental runs, also include
	// recently-closed issues so the store can be updated when an issue closes.
	if since.IsZero() {
		fmt.Fprintf(&sb, "involves:%s is:issue is:open", s.config.User)
	} else {
		fmt.Fprintf(&sb, "involves:%s is:issue updated:>%s", s.config.User, since.UTC().Format("2006-01-02"))
	}
	for _, r := range s.config.Repos {
		fmt.Fprintf(&sb, " repo:%s", r)
	}
	searchQuery := sb.String()

	var allItems []core.Item
	var cursor *string
	remaining := maxIssues

	for remaining > 0 {
		pageSize := min(remaining, maxPerPage)
		vars := map[string]any{
			"q":            searchQuery,
			"first":        pageSize,
			"after":        cursor, // nil on first page
			"commentLimit": commentLimit,
		}

		data, err := doGraphQL[involvedIssuesResponse](ctx, s.config.Token, s.graphqlEndpoint(), involvedIssuesQuery, vars)
		if err != nil {
			return nil, fmt.Errorf("fetch involved issues: %w", err)
		}

		search := data.Search
		for _, node := range search.Nodes {
			// Search can return non-issue results; skip anything that didn't
			// match the Issue fragment (Number will be zero).
			if node.Number == 0 {
				continue
			}
			allItems = append(allItems, s.normalizeIssue(node.issueFields, node.Repository.NameWithOwner))
			remaining--
			if remaining == 0 {
				break
			}
		}

		if !search.PageInfo.HasNextPage || remaining == 0 {
			break
		}
		c := search.PageInfo.EndCursor
		cursor = &c
	}

	return allItems, nil
}

// normalizeIssue maps a raw GitHub issue onto a core.Item. namespace is the
// "owner/repo" string for the issue.
func (s *Source) normalizeIssue(issue issueFields, namespace string) core.Item {
	cfg := s.config
	user := cfg.User

	comments := issue.commentEntries()

	labels := make([]string, 0, len(issue.Labels.Nodes))
	for _, l := range issue.Labels.Nodes {
		labels = append(labels, l.Name)
	}
	attrs := IssueAttributes{
		Author: issue.Author.Login,
		Labels: labels,
		State:  issue.State,
	}

	// Closed issues are never actionable. Return a tombstone so the runner
	// removes any stale entry from the store.
	if issue.State == "CLOSED" {
		return core.Item{
			ID:     ItemID(namespace, issue.Number),
			Closed: true,
		}
	}

	// IsAssigned — reviewer signal is not applicable to issues (silently skipped).
	assignedSignals := cfg.IsAssigned
	if len(assignedSignals) == 0 {
		assignedSignals = []AssignedSignal{AssignedSignalAuthor, AssignedSignalAssignee}
	}
	isAssigned := false
	for _, sig := range assignedSignals {
		switch sig {
		case AssignedSignalAuthor:
			if issue.Author.Login == user {
				isAssigned = true
			}
		case AssignedSignalAssignee:
			for _, a := range issue.Assignees.Nodes {
				if a.Login == user {
					isAssigned = true
				}
			}
		// AssignedSignalReviewer: not applicable to issues, silently skip.
		}
	}

	// WaitsOnMe — evaluate applicable signals using comments only (no reviews
	// or commits on issues). PR-only signals (approved, approved_not_merged)
	// are silently skipped.
	waitsSignals := cfg.WaitsOnMe
	if len(waitsSignals) == 0 {
		waitsSignals = []WaitsOnMeSignal{
			WaitsOnMeUnreviewed,
			WaitsOnMeAuthorUpdated,
			WaitsOnMePeerActivity,
			WaitsOnMeReviewReceived,
		}
	}
	staleDays := cfg.StaleDays
	if staleDays == 0 {
		staleDays = defaultStaleDays
	}
	waitsOnMe := false
	var activeSignals []string
	for _, sig := range waitsSignals {
		fired := false
		switch sig {
		case WaitsOnMeUnreviewed:
			// Issue is not mine and I have never commented on it.
			if issue.Author.Login != user && latestActivityBy(nil, comments, user) == nil {
				fired = true
			}
		case WaitsOnMeAuthorUpdated:
			// I've commented before, and the author has since replied.
			if issue.Author.Login != user {
				if myLast := latestActivityBy(nil, comments, user); myLast != nil {
					if authorLatest := latestActivityBy(nil, comments, issue.Author.Login); authorLatest != nil && authorLatest.After(*myLast) {
						fired = true
					}
				}
			}
		case WaitsOnMePeerActivity:
			// Someone who is neither the author nor me has commented since my
			// last activity (or at any time if I've never engaged).
			myLast := latestActivityBy(nil, comments, user)
			peerLatest := latestActivityExcluding(nil, comments, user, issue.Author.Login)
			if peerLatest != nil && (myLast == nil || peerLatest.After(*myLast)) {
				fired = true
			}
		case WaitsOnMeReviewReceived:
			// I opened this issue and someone else has commented since my last
			// comment (or since creation if I haven't commented yet).
			if issue.Author.Login == user {
				myLastUpdate := latestTime(issue.CreatedAt, latestActivityBy(nil, comments, user))
				if t := latestActivityExcluding(nil, comments, user); t != nil && t.After(myLastUpdate) {
					fired = true
				}
			}
		case WaitsOnMeStale:
			if time.Since(issue.UpdatedAt) >= time.Duration(staleDays)*24*time.Hour {
				fired = true
			}
		// WaitsOnMeApproved, WaitsOnMeApprovedNotMerged: PR-only, skip for issues.
		}
		if fired {
			waitsOnMe = true
			activeSignals = append(activeSignals, string(sig))
		}
	}
	attrs.ActiveSignals = activeSignals

	// UserActivityAt — most recent comment by the user on this issue.
	// Review-type interactions (approve, request_changes) don't exist on issues,
	// so comments are always tracked regardless of the Interactions config.
	var userActivityAt *time.Time
	for _, c := range comments {
		if c.Login == user {
			t := c.CreatedAt
			if userActivityAt == nil || t.After(*userActivityAt) {
				userActivityAt = &t
			}
		}
	}

	attrsJSON, _ := json.Marshal(attrs)

	return core.Item{
		ID:             ItemID(namespace, issue.Number),
		Source:         Kind,
		Type:           TypeIssue,
		Title:          issue.Title,
		URL:            issue.URL,
		Namespace:      namespace,
		CreatedAt:      issue.CreatedAt,
		UpdatedAt:      issue.UpdatedAt,
		WaitsOnMe:      waitsOnMe,
		IsAssigned:     isAssigned,
		UserActivityAt: userActivityAt,
		Attributes:     attrsJSON,
	}
}
