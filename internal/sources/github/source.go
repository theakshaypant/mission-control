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
	name         string
	config       *Config
	lastSyncedAt *time.Time
}

func New(name string, cfg *Config) *Source {
	return &Source{name: name, config: cfg}
}

func (s *Source) Name() string              { return s.name }
func (s *Source) Kind() core.SourceKind     { return Kind }
func (s *Source) Config() core.SourceConfig { return s.config }
func (s *Source) LastSyncedAt() *time.Time  { return s.lastSyncedAt }

// graphqlEndpoint returns the GitHub GraphQL API URL for this source.
// Defaults to the public GitHub endpoint; uses the GHE path for custom hosts.
func (s *Source) graphqlEndpoint() string {
	if s.config.Host == "" || s.config.Host == "github.com" {
		return "https://api.github.com/graphql"
	}
	return "https://" + s.config.Host + "/api/graphql"
}

// Sync fetches GitHub PRs from all configured repos since the last sync.
func (s *Source) Sync(ctx context.Context) ([]core.Item, error) {
	items, err := s.syncPRs(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	s.lastSyncedAt = &now
	return items, nil
}
