package github

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/theakshaypant/mission-control/internal/core"
)

const (
	defaultMaxPRs    = 50
	maxPerPage       = 100
	defaultStaleDays = 30
)

// reviewEntry is a normalized review event used by activity helpers.
type reviewEntry struct {
	Login       string
	State       string
	SubmittedAt time.Time
}

// commentEntry is a normalized comment event used by activity helpers.
type commentEntry struct {
	Login     string
	CreatedAt time.Time
}

// prFields contains all GraphQL fields fetched for a pull request, shared
// between the per-repo query and the search query.
type prFields struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	URL       string    `json:"url"`
	IsDraft   bool      `json:"isDraft"`
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
	ReviewRequests struct {
		Nodes []struct {
			RequestedReviewer struct {
				Login string `json:"login"` // empty for team reviewers
			} `json:"requestedReviewer"`
		} `json:"nodes"`
	} `json:"reviewRequests"`
	Reviews struct {
		Nodes []struct {
			Author      struct{ Login string `json:"login"` } `json:"author"`
			State       string                                `json:"state"`
			SubmittedAt time.Time                             `json:"submittedAt"`
		} `json:"nodes"`
	} `json:"reviews"`
	Comments struct {
		Nodes []struct {
			Author    struct{ Login string `json:"login"` } `json:"author"`
			CreatedAt time.Time                             `json:"createdAt"`
		} `json:"nodes"`
	} `json:"comments"`
	// State is the PR's lifecycle state: "OPEN", "MERGED", or "CLOSED".
	State string `json:"state"`
	// ReviewDecision is GitHub's computed review state for this PR.
	// Values: "APPROVED", "CHANGES_REQUESTED", "REVIEW_REQUIRED", or "".
	ReviewDecision string `json:"reviewDecision"`
	// Commits fetches the most recent commits to detect the latest push date.
	Commits struct {
		Nodes []struct {
			Commit struct {
				PushedDate    *time.Time `json:"pushedDate"`
				CommittedDate time.Time  `json:"committedDate"`
			} `json:"commit"`
		} `json:"nodes"`
	} `json:"commits"`
}

func (pr prFields) reviewEntries() []reviewEntry {
	result := make([]reviewEntry, 0, len(pr.Reviews.Nodes))
	for _, r := range pr.Reviews.Nodes {
		result = append(result, reviewEntry{Login: r.Author.Login, State: r.State, SubmittedAt: r.SubmittedAt})
	}
	return result
}

func (pr prFields) commentEntries() []commentEntry {
	result := make([]commentEntry, 0, len(pr.Comments.Nodes))
	for _, c := range pr.Comments.Nodes {
		result = append(result, commentEntry{Login: c.Author.Login, CreatedAt: c.CreatedAt})
	}
	return result
}

// prsByRepoQuery fetches all open PRs in a single repo, ordered by most
// recently updated. Supports cursor-based pagination.
const prsByRepoQuery = `
query PRsByRepo($owner: String!, $repo: String!, $first: Int!, $after: String) {
  repository(owner: $owner, name: $repo) {
    pullRequests(states: [OPEN], first: $first, after: $after, orderBy: {field: UPDATED_AT, direction: DESC}) {
      pageInfo { hasNextPage endCursor }
      nodes {
        number title url isDraft state createdAt updatedAt reviewDecision
        author { login }
        assignees(first: 10) { nodes { login } }
        labels(first: 20) { nodes { name } }
        reviewRequests(first: 10) {
          nodes { requestedReviewer { ... on User { login } } }
        }
        reviews(last: 100) {
          nodes { author { login } state submittedAt }
        }
        comments(last: 100) {
          nodes { author { login } createdAt }
        }
        commits(last: 1) {
          nodes { commit { pushedDate committedDate } }
        }
      }
    }
  }
}`

type prsByRepoResponse struct {
	Repository struct {
		PullRequests struct {
			PageInfo struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
			Nodes []prFields `json:"nodes"`
		} `json:"pullRequests"`
	} `json:"repository"`
}

// involvedPRsQuery uses GitHub's search API to fetch open PRs across all
// configured repos where the user is involved (author, reviewer, assignee).
const involvedPRsQuery = `
query InvolvedPRs($q: String!, $first: Int!, $after: String) {
  search(query: $q, type: ISSUE, first: $first, after: $after) {
    pageInfo { hasNextPage endCursor }
    nodes {
      ... on PullRequest {
        number title url isDraft state createdAt updatedAt reviewDecision
        repository { nameWithOwner }
        author { login }
        assignees(first: 10) { nodes { login } }
        labels(first: 20) { nodes { name } }
        reviewRequests(first: 10) {
          nodes { requestedReviewer { ... on User { login } } }
        }
        reviews(last: 100) {
          nodes { author { login } state submittedAt }
        }
        comments(last: 100) {
          nodes { author { login } createdAt }
        }
        commits(last: 1) {
          nodes { commit { pushedDate committedDate } }
        }
      }
    }
  }
}`

// closedPRsByRepoQuery fetches recently merged or closed PRs in a single repo,
// used on incremental syncs to produce tombstones for PRs that are no longer open.
const closedPRsByRepoQuery = `
query ClosedPRsByRepo($owner: String!, $repo: String!, $first: Int!, $after: String) {
  repository(owner: $owner, name: $repo) {
    pullRequests(states: [MERGED, CLOSED], first: $first, after: $after, orderBy: {field: UPDATED_AT, direction: DESC}) {
      pageInfo { hasNextPage endCursor }
      nodes { number state updatedAt }
    }
  }
}`

type closedPRsByRepoResponse struct {
	Repository struct {
		PullRequests struct {
			PageInfo struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
			Nodes []struct {
				Number    int       `json:"number"`
				State     string    `json:"state"`
				UpdatedAt time.Time `json:"updatedAt"`
			} `json:"nodes"`
		} `json:"pullRequests"`
	} `json:"repository"`
}

type involvedPRsResponse struct {
	Search struct {
		PageInfo struct {
			HasNextPage bool   `json:"hasNextPage"`
			EndCursor   string `json:"endCursor"`
		} `json:"pageInfo"`
		Nodes []struct {
			prFields
			Repository struct {
				NameWithOwner string `json:"nameWithOwner"`
			} `json:"repository"`
		} `json:"nodes"`
	} `json:"search"`
}

func (s *Source) syncPRs(ctx context.Context, sincePtr *time.Time) ([]core.Item, error) {
	scope := s.config.PRScope
	if scope == "" {
		scope = FetchScopeInvolved
	}
	maxPRs := s.config.MaxPRs
	if maxPRs == 0 {
		maxPRs = defaultMaxPRs
	}

	var since time.Time
	if sincePtr != nil {
		since = *sincePtr
	}

	var items []core.Item
	var err error
	switch scope {
	case FetchScopeAll:
		items, err = s.fetchAllPRs(ctx, since, maxPRs)
	case FetchScopeInvolved:
		items, err = s.fetchInvolvedPRs(ctx, since, maxPRs)
	default:
		return nil, fmt.Errorf("github: unknown pr_scope %q", scope)
	}
	if err != nil {
		return nil, err
	}

	return items, nil
}

// fetchAllPRs retrieves all open PRs from each configured repo. Results are
// ordered by most recently updated; pagination stops once all remaining PRs
// predate the since time or MaxPRs is reached per repo.
func (s *Source) fetchAllPRs(ctx context.Context, since time.Time, maxPRs int) ([]core.Item, error) {
	var allItems []core.Item

	for _, repoStr := range s.config.Repos {
		parts := strings.SplitN(repoStr, "/", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("github: invalid repo format %q", repoStr)
		}
		owner, repo := parts[0], parts[1]

		remaining := maxPRs
		var cursor *string
		for remaining > 0 {
			pageSize := min(remaining, maxPerPage)
			vars := map[string]any{
				"owner": owner,
				"repo":  repo,
				"first": pageSize,
				"after": cursor, // nil on first page
			}

			data, err := doGraphQL[prsByRepoResponse](ctx, s.config.Token, s.graphqlEndpoint(), prsByRepoQuery, vars)
			if err != nil {
				return nil, fmt.Errorf("fetch PRs for %s: %w", repoStr, err)
			}

			prs := data.Repository.PullRequests
			done := false
			for _, pr := range prs.Nodes {
				// Since results are ordered DESC by updatedAt, once we see a
				// PR older than our last sync we can stop fetching this repo.
				if !since.IsZero() && !pr.UpdatedAt.After(since) {
					done = true
					break
				}
				allItems = append(allItems, s.normalizePR(pr, repoStr))
				remaining--
				if remaining == 0 {
					done = true
					break
				}
			}

			if done || !prs.PageInfo.HasNextPage {
				break
			}
			c := prs.PageInfo.EndCursor
			cursor = &c
		}

		// On incremental syncs, also fetch recently merged/closed PRs so we can
		// tombstone any that are still in the store.
		if !since.IsZero() {
			closed, err := s.fetchClosedPRsForRepo(ctx, owner, repo, since)
			if err != nil {
				return nil, fmt.Errorf("fetch closed PRs for %s: %w", repoStr, err)
			}
			allItems = append(allItems, closed...)
		}
	}

	return allItems, nil
}

// fetchClosedPRsForRepo returns tombstone items for PRs in owner/repo that were
// merged or closed after since.
func (s *Source) fetchClosedPRsForRepo(ctx context.Context, owner, repo string, since time.Time) ([]core.Item, error) {
	var items []core.Item
	var cursor *string
	for {
		vars := map[string]any{
			"owner": owner,
			"repo":  repo,
			"first": maxPerPage,
			"after": cursor,
		}
		data, err := doGraphQL[closedPRsByRepoResponse](ctx, s.config.Token, s.graphqlEndpoint(), closedPRsByRepoQuery, vars)
		if err != nil {
			return nil, err
		}
		prs := data.Repository.PullRequests
		done := false
		for _, pr := range prs.Nodes {
			if !pr.UpdatedAt.After(since) {
				done = true
				break
			}
			items = append(items, core.Item{
				ID:     ItemID(owner+"/"+repo, pr.Number),
				Closed: true,
			})
		}
		if done || !prs.PageInfo.HasNextPage {
			break
		}
		c := prs.PageInfo.EndCursor
		cursor = &c
	}
	return items, nil
}

// fetchInvolvedPRs uses GitHub's search API to fetch open PRs across all
// configured repos where the user is involved. A since date filter is added
// when available to limit results to recently updated PRs.
func (s *Source) fetchInvolvedPRs(ctx context.Context, since time.Time, maxPRs int) ([]core.Item, error) {
	var sb strings.Builder
	// On the first run (no cursor) restrict to open PRs to avoid pulling in
	// historical merged/closed PRs. On incremental runs, also include recently
	// merged/closed PRs so the store can be tombstoned when a PR closes.
	if since.IsZero() {
		fmt.Fprintf(&sb, "involves:%s is:pr is:open", s.config.User)
	} else {
		fmt.Fprintf(&sb, "involves:%s is:pr updated:>%s", s.config.User, since.UTC().Format("2006-01-02"))
	}
	for _, r := range s.config.Repos {
		fmt.Fprintf(&sb, " repo:%s", r)
	}
	searchQuery := sb.String()

	var allItems []core.Item
	var cursor *string
	remaining := maxPRs

	for remaining > 0 {
		pageSize := min(remaining, maxPerPage)
		vars := map[string]any{
			"q":     searchQuery,
			"first": pageSize,
			"after": cursor, // nil on first page
		}

		data, err := doGraphQL[involvedPRsResponse](ctx, s.config.Token, s.graphqlEndpoint(), involvedPRsQuery, vars)
		if err != nil {
			return nil, fmt.Errorf("fetch involved PRs: %w", err)
		}

		search := data.Search
		for _, node := range search.Nodes {
			// Search can return non-PR results; skip anything that didn't
			// match the PullRequest fragment (Number will be zero).
			if node.Number == 0 {
				continue
			}
			allItems = append(allItems, s.normalizePR(node.prFields, node.Repository.NameWithOwner))
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

// normalizePR maps a raw GitHub PR onto a core.Item. namespace is the
// "owner/repo" string for the PR.
func (s *Source) normalizePR(pr prFields, namespace string) core.Item {
	// Merged or closed PRs are tombstones: signal the runner to remove any
	// stale entry from the store so they stop appearing on the dashboard.
	if pr.State == "MERGED" || pr.State == "CLOSED" {
		return core.Item{
			ID:     ItemID(namespace, pr.Number),
			Closed: true,
		}
	}

	cfg := s.config
	user := cfg.User

	reviews := pr.reviewEntries()
	comments := pr.commentEntries()

	labels := make([]string, 0, len(pr.Labels.Nodes))
	for _, l := range pr.Labels.Nodes {
		labels = append(labels, l.Name)
	}
	attrs := PRAttributes{
		Author:         pr.Author.Login,
		Labels:         labels,
		IsDraft:        pr.IsDraft,
		State:          "open",
		ReviewDecision: pr.ReviewDecision,
	}

	// IsAssigned — default to all three conditions when none are configured.
	assignedSignals := cfg.IsAssigned
	if len(assignedSignals) == 0 {
		assignedSignals = []AssignedSignal{AssignedSignalAuthor, AssignedSignalAssignee, AssignedSignalReviewer}
	}
	isAssigned := false
	for _, sig := range assignedSignals {
		switch sig {
		case AssignedSignalAuthor:
			if pr.Author.Login == user {
				isAssigned = true
			}
		case AssignedSignalAssignee:
			for _, a := range pr.Assignees.Nodes {
				if a.Login == user {
					isAssigned = true
				}
			}
		case AssignedSignalReviewer:
			for _, rr := range pr.ReviewRequests.Nodes {
				if rr.RequestedReviewer.Login == user {
					isAssigned = true
				}
			}
		}
	}

	// WaitsOnMe — default to the four most broadly useful signals.
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
			// PR is not mine and I have never reviewed or commented on it.
			if pr.Author.Login != user && latestActivityBy(reviews, comments, user) == nil {
				fired = true
			}
		case WaitsOnMeAuthorUpdated:
			// I've reviewed before, and the author has since pushed commits or replied.
			if pr.Author.Login != user {
				if myLast := latestActivityBy(reviews, comments, user); myLast != nil {
					authorLatest := latestTime(time.Time{}, latestCommit(pr), latestActivityBy(reviews, comments, pr.Author.Login))
					if !authorLatest.IsZero() && authorLatest.After(*myLast) {
						fired = true
					}
				}
			}
		case WaitsOnMePeerActivity:
			// Someone who is neither the author nor me has reviewed or commented
			// since my last activity (or at any time if I've never engaged).
			// Skipped when the user is the author — review_received covers that case.
			if pr.Author.Login != user {
				myLast := latestActivityBy(reviews, comments, user)
				peerLatest := latestActivityExcluding(reviews, comments, user, pr.Author.Login)
				if peerLatest != nil && (myLast == nil || peerLatest.After(*myLast)) {
					fired = true
				}
			}
		case WaitsOnMeApprovedNotMerged:
			// GitHub considers this PR fully approved.
			if pr.ReviewDecision == "APPROVED" {
				fired = true
			}
		case WaitsOnMeReviewReceived:
			// I'm the author and someone else has commented or reviewed since
			// my last commit push or comment.
			if pr.Author.Login == user {
				myLastUpdate := latestTime(pr.CreatedAt, latestCommit(pr), latestActivityBy(reviews, comments, user))
				if t := latestActivityExcluding(reviews, comments, user); t != nil && t.After(myLastUpdate) {
					fired = true
				}
			}
		case WaitsOnMeApproved:
			// I'm the author and GitHub considers the PR approved.
			if pr.Author.Login == user && pr.ReviewDecision == "APPROVED" {
				fired = true
			}
		case WaitsOnMeStale:
			if time.Since(pr.UpdatedAt) >= time.Duration(staleDays)*24*time.Hour {
				fired = true
			}
		}
		if fired {
			waitsOnMe = true
			activeSignals = append(activeSignals, string(sig))
		}
	}
	attrs.ActiveSignals = activeSignals

	// UserActivityAt — most recent configured interaction by the user.
	interactions := interactionSet(cfg)
	var userActivityAt *time.Time

	for _, r := range reviews {
		if r.Login != user {
			continue
		}
		var counts bool
		switch r.State {
		case "APPROVED":
			_, isApprove := interactions[InteractionApprove]
			_, isReview := interactions[InteractionReview]
			counts = isApprove || isReview
		case "CHANGES_REQUESTED":
			_, isRC := interactions[InteractionRequestChanges]
			_, isReview := interactions[InteractionReview]
			counts = isRC || isReview
		default: // COMMENTED, DISMISSED, PENDING
			_, counts = interactions[InteractionReview]
		}
		if counts {
			t := r.SubmittedAt
			if userActivityAt == nil || t.After(*userActivityAt) {
				userActivityAt = &t
			}
		}
	}

	if _, ok := interactions[InteractionComment]; ok {
		for _, c := range comments {
			if c.Login == user {
				t := c.CreatedAt
				if userActivityAt == nil || t.After(*userActivityAt) {
					userActivityAt = &t
				}
			}
		}
	}

	attrsJSON, _ := json.Marshal(attrs)

	return core.Item{
		ID:             ItemID(namespace, pr.Number),
		Source:         Kind,
		Type:           TypePR,
		Title:          pr.Title,
		URL:            pr.URL,
		Namespace:      namespace,
		CreatedAt:      pr.CreatedAt,
		UpdatedAt:      pr.UpdatedAt,
		WaitsOnMe:      waitsOnMe,
		IsAssigned:     isAssigned,
		UserActivityAt: userActivityAt,
		Attributes:     attrsJSON,
	}
}

// latestActivityBy returns the most recent review or comment timestamp by the
// given login, or nil if they have no recorded activity.
func latestActivityBy(reviews []reviewEntry, comments []commentEntry, login string) *time.Time {
	var latest *time.Time
	for _, r := range reviews {
		if r.Login == login {
			t := r.SubmittedAt
			if latest == nil || t.After(*latest) {
				latest = &t
			}
		}
	}
	for _, c := range comments {
		if c.Login == login {
			t := c.CreatedAt
			if latest == nil || t.After(*latest) {
				latest = &t
			}
		}
	}
	return latest
}

// latestActivityExcluding returns the most recent review or comment timestamp
// by anyone whose login is not in the excluded set, or nil if there is none.
func latestActivityExcluding(reviews []reviewEntry, comments []commentEntry, excludeLogins ...string) *time.Time {
	excluded := make(map[string]bool, len(excludeLogins))
	for _, l := range excludeLogins {
		excluded[l] = true
	}
	var latest *time.Time
	for _, r := range reviews {
		if !excluded[r.Login] {
			t := r.SubmittedAt
			if latest == nil || t.After(*latest) {
				latest = &t
			}
		}
	}
	for _, c := range comments {
		if !excluded[c.Login] {
			t := c.CreatedAt
			if latest == nil || t.After(*latest) {
				latest = &t
			}
		}
	}
	return latest
}

// latestCommit returns the pushed date (or committed date when push date is
// absent) of the most recently fetched commit, or nil if there are none.
// The query requests commits(last: 1) so there is at most one node.
func latestCommit(pr prFields) *time.Time {
	if len(pr.Commits.Nodes) == 0 {
		return nil
	}
	c := pr.Commits.Nodes[0].Commit
	if c.PushedDate != nil {
		return c.PushedDate
	}
	t := c.CommittedDate
	return &t
}

// latestTime returns the latest of a base time and any number of optional
// times, ignoring nil pointers.
func latestTime(base time.Time, candidates ...*time.Time) time.Time {
	result := base
	for _, t := range candidates {
		if t != nil && t.After(result) {
			result = *t
		}
	}
	return result
}

// interactionSet returns a set of configured interaction types for fast lookup.
// If none are configured, all interaction types are included.
func interactionSet(cfg *Config) map[Interaction]struct{} {
	interactions := cfg.Interactions
	if len(interactions) == 0 {
		interactions = []Interaction{
			InteractionReview,
			InteractionComment,
			InteractionApprove,
			InteractionRequestChanges,
		}
	}
	set := make(map[Interaction]struct{}, len(interactions))
	for _, i := range interactions {
		set[i] = struct{}{}
	}
	return set
}
