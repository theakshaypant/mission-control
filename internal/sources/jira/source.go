package jira

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/theakshaypant/mission-control/internal/core"
	"github.com/theakshaypant/mission-control/internal/sources"
)

func init() {
	sources.Register(string(Kind), func(name string, raw map[string]any) (core.Source, error) {
		var cfg Config
		if err := sources.UnmarshalRaw(raw, &cfg); err != nil {
			return nil, fmt.Errorf("jira config: %w", err)
		}
		if err := cfg.Validate(); err != nil {
			return nil, err
		}
		return New(name, &cfg), nil
	})
}

// Source implements core.Source for Jira Cloud.
type Source struct {
	name   string
	config *Config
}

func New(name string, cfg *Config) *Source {
	return &Source{name: name, config: cfg}
}

func (s *Source) Name() string              { return s.name }
func (s *Source) Kind() core.SourceKind     { return Kind }
func (s *Source) Config() core.SourceConfig { return s.config }

func (s *Source) apiVersion() int {
	if s.config.APIVersion != 0 {
		return s.config.APIVersion
	}
	return 3
}

// Sync fetches Jira tickets from all configured boards. since is the cursor
// from the last successful sync; pass nil for a full fetch on first run.
// Tickets that appear in multiple boards are deduplicated by issue key: the
// first board's namespace is preserved and active signals are unioned.
func (s *Source) Sync(ctx context.Context, since *time.Time) ([]core.Item, error) {
	// index maps item ID → position in allItems for O(1) dedup lookups.
	var allItems []core.Item
	index := make(map[string]int)

	for _, board := range s.config.Boards {
		items, err := s.syncBoard(ctx, board, since)
		if err != nil {
			return nil, fmt.Errorf("jira: sync board %q: %w", board.Name, err)
		}
		for _, item := range items {
			if idx, ok := index[item.ID]; ok {
				allItems[idx] = mergeItems(allItems[idx], item)
			} else {
				index[item.ID] = len(allItems)
				allItems = append(allItems, item)
			}
		}
	}

	// On incremental syncs, fetch recently-resolved tickets across the instance
	// so any stored items that moved to a done status can be tombstoned.
	if since != nil {
		tombstones, err := s.fetchTombstones(ctx, *since)
		if err != nil {
			return nil, fmt.Errorf("jira: fetch tombstones: %w", err)
		}
		for _, t := range tombstones {
			if idx, ok := index[t.ID]; ok {
				allItems[idx] = t
			} else {
				index[t.ID] = len(allItems)
				allItems = append(allItems, t)
			}
		}
	}

	return allItems, nil
}

// syncBoard fetches tickets for a single board. On incremental syncs a time
// filter is injected into the JQL so only recently updated tickets are fetched.
func (s *Source) syncBoard(ctx context.Context, board Board, since *time.Time) ([]core.Item, error) {
	maxResults := board.MaxResults
	if maxResults == 0 {
		maxResults = defaultMaxResults
	}

	jql := board.JQL
	if since != nil {
		jql = addUpdatedFilter(board.JQL, *since)
	}

	var items []core.Item
	err := s.search(ctx, jql, maxResults, func(issues []issueNode) error {
		for _, issue := range issues {
			items = append(items, s.normalizeTicket(issue, board.Name, since))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return items, nil
}

// fetchTombstones returns tombstone items for tickets that moved into a done
// status since the given time. It runs one query across the entire Jira instance;
// the runner will call DeleteItem only for IDs already in the store, so extra
// results are harmless.
func (s *Source) fetchTombstones(ctx context.Context, since time.Time) ([]core.Item, error) {
	done := s.doneStatuses()
	if len(done) == 0 {
		return nil, nil
	}

	quoted := make([]string, len(done))
	for i, status := range done {
		quoted[i] = `"` + status + `"`
	}
	jql := fmt.Sprintf(
		`status in (%s) AND updated > "%s"`,
		strings.Join(quoted, ", "),
		since.UTC().Format("2006-01-02 15:04"),
	)

	var tombstones []core.Item
	err := s.search(ctx, jql, 100, func(issues []issueNode) error {
		for _, issue := range issues {
			tombstones = append(tombstones, core.Item{
				ID:     ItemID(s.name, issue.Key),
				Closed: true,
			})
		}
		return nil
	})
	return tombstones, err
}
