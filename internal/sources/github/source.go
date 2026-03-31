package github

import (
	"context"
	"fmt"
	"time"

	"github.com/theakshaypant/mission-control/internal/core"
	"github.com/theakshaypant/mission-control/internal/sources"
)

func init() {
	sources.Register(string(Kind), func(name string, raw map[string]any) (core.Source, error) {
		var cfg Config
		if err := sources.UnmarshalRaw(raw, &cfg); err != nil {
			return nil, fmt.Errorf("github config: %w", err)
		}
		if err := cfg.Validate(); err != nil {
			return nil, err
		}
		return New(name, &cfg), nil
	})
}

// Source implements core.Source for GitHub.
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

// graphqlEndpoint returns the GitHub GraphQL API URL for this source.
// Defaults to the public GitHub endpoint; uses the GHE path for custom hosts.
func (s *Source) graphqlEndpoint() string {
	if s.config.Host == "" || s.config.Host == "github.com" {
		return "https://api.github.com/graphql"
	}
	return "https://" + s.config.Host + "/api/graphql"
}

// Sync fetches GitHub items from all configured repos. since is the cursor
// from the last successful sync; pass nil for a full fetch on first run.
// PRs are fetched unless pr_scope is "none". Issues are fetched only when
// issue_scope is set to "involved" or "all".
func (s *Source) Sync(ctx context.Context, since *time.Time) ([]core.Item, error) {
	var items []core.Item

	if s.config.PRScope != FetchScopeNone {
		prItems, err := s.syncPRs(ctx, since)
		if err != nil {
			return nil, err
		}
		items = append(items, prItems...)
	}

	if s.config.IssueScope != "" && s.config.IssueScope != FetchScopeNone {
		issueItems, err := s.syncIssues(ctx, since)
		if err != nil {
			return nil, err
		}
		items = append(items, issueItems...)
	}

	return items, nil
}
