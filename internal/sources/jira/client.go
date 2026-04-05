package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)


// requestFields lists the Jira issue fields requested on every search call.
var requestFields = []string{
	"summary", "status", "assignee", "reporter",
	"issuetype", "priority", "labels", "created", "updated", "comment",
}

// jiraTime wraps time.Time to handle Jira Cloud's timestamp format:
// "2006-01-02T15:04:05.000+0000" (ISO 8601 with milliseconds, no colon in offset).
type jiraTime struct {
	time.Time
}

func (jt *jiraTime) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	if s == "" || s == "null" {
		return nil
	}
	t, err := time.Parse("2006-01-02T15:04:05.000-0700", s)
	if err != nil {
		return fmt.Errorf("jira: parse time %q: %w", s, err)
	}
	jt.Time = t
	return nil
}

// searchRequest is the POST body for POST /rest/api/3/search/jql.
// expand is a comma-separated string per the Jira Cloud REST API v3 spec.
type searchRequest struct {
	JQL           string   `json:"jql"`
	Fields        []string `json:"fields"`
	Expand        string   `json:"expand,omitempty"`
	MaxResults    int      `json:"maxResults"`
	NextPageToken string   `json:"nextPageToken,omitempty"`
}

// searchResponse is the response from POST /rest/api/3/search/jql.
// Pagination is cursor-based: NextPageToken is absent when there are no more pages.
type searchResponse struct {
	Issues        []issueNode `json:"issues"`
	NextPageToken string      `json:"nextPageToken"`
}

// issueNode is one issue returned by the search API.
type issueNode struct {
	Key       string      `json:"key"`
	Fields    issueFields `json:"fields"`
	Changelog changelog   `json:"changelog"`
}

// issueFields holds the subset of Jira fields we request.
type issueFields struct {
	Summary  string `json:"summary"`
	Status   struct {
		Name string `json:"name"`
	} `json:"status"`
	Assignee  *userField `json:"assignee"`
	Reporter  *userField `json:"reporter"`
	IssueType struct {
		Name string `json:"name"`
	} `json:"issuetype"`
	Priority *struct {
		Name string `json:"name"`
	} `json:"priority"`
	Labels  []string `json:"labels"`
	Created jiraTime `json:"created"`
	Updated jiraTime `json:"updated"`
	Comment struct {
		Comments []commentNode `json:"comments"`
	} `json:"comment"`
}

// userField represents a Jira user reference embedded in issue fields.
type userField struct {
	EmailAddress string `json:"emailAddress"`
	DisplayName  string `json:"displayName"`
}

// commentNode is a single comment on an issue.
type commentNode struct {
	Author  userField `json:"author"`
	Created jiraTime  `json:"created"`
}

// changelog holds the full history of field changes for an issue, returned
// when the search is called with expand=changelog.
type changelog struct {
	Histories []changeHistory `json:"histories"`
}

// changeHistory is one batch of field changes made at a single point in time.
type changeHistory struct {
	Author  userField    `json:"author"`
	Created jiraTime     `json:"created"`
	Items   []changeItem `json:"items"`
}

// changeItem describes a single field change within a changeHistory.
type changeItem struct {
	Field      string `json:"field"`
	FromString string `json:"fromString"`
	ToString   string `json:"toString"`
}

// search executes a paginated JQL search against the Jira REST API v3, calling
// fn for each page of results. Uses POST /rest/api/3/search/jql with
// cursor-based pagination (nextPageToken). Stops when fn returns an error,
// maxResults tickets have been fetched, or the API signals no more pages.
func (s *Source) search(ctx context.Context, jql string, maxResults int, fn func([]issueNode) error) error {
	endpoint := fmt.Sprintf("https://%s/rest/api/%d/search/jql", s.config.Host, s.apiVersion())
	fetched := 0
	var nextPageToken string

	for {
		pageSize := min(maxResults-fetched, 100)
		issues, token, err := s.searchPage(ctx, endpoint, jql, pageSize, nextPageToken)
		if err != nil {
			return err
		}
		if len(issues) == 0 {
			break
		}
		if err := fn(issues); err != nil {
			return err
		}
		fetched += len(issues)
		if fetched >= maxResults || token == "" {
			break
		}
		nextPageToken = token
	}
	return nil
}

// searchPage executes one POST /rest/api/3/search/jql request and returns the
// issues, the next page token (empty string when no more pages exist), and any error.
func (s *Source) searchPage(ctx context.Context, endpoint, jql string, pageSize int, nextPageToken string) ([]issueNode, string, error) {
	body := searchRequest{
		JQL:           jql,
		Fields:        requestFields,
		Expand:        "changelog",
		MaxResults:    pageSize,
		NextPageToken: nextPageToken,
	}
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, "", fmt.Errorf("jira: marshal search request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, "", fmt.Errorf("jira: create request: %w", err)
	}
	req.SetBasicAuth(s.config.Email, s.config.Token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("jira: do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, "", fmt.Errorf("jira: api returned status %d for JQL %q: %s",
			resp.StatusCode, truncateJQL(jql), strings.TrimSpace(string(body)))
	}

	var sr searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, "", fmt.Errorf("jira: decode search response: %w", err)
	}
	return sr.Issues, sr.NextPageToken, nil
}

// truncateJQL shortens a JQL string for use in error messages.
func truncateJQL(jql string) string {
	jql = strings.TrimSpace(jql)
	if len(jql) <= 80 {
		return jql
	}
	return jql[:79] + "…"
}
