// Package app provides the dependency container for mission-control.
// It wires config, store, sources, sync runner, and the actions layer
// into a single App value shared by both the CLI and the API server.
package app

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/theakshaypant/mission-control/internal/actions"
	"github.com/theakshaypant/mission-control/internal/config"
	"github.com/theakshaypant/mission-control/internal/sources"
	"github.com/theakshaypant/mission-control/internal/store/jsonfile"
	syncp "github.com/theakshaypant/mission-control/internal/sync"

	// Register source factories via init().
	_ "github.com/theakshaypant/mission-control/internal/sources/github"
)

// App wires all mission-control dependencies together. Set ConfigPath before
// calling Init; all other fields are populated by Init.
type App struct {
	// ConfigPath is the path to config.yaml.
	// If empty, DefaultConfigPath is used.
	ConfigPath string

	// Config is populated by Init.
	Config *config.AppConfig

	// Actions is the shared service layer, available after Init.
	Actions *actions.Actions

	// Scheduler runs background syncs. Start it with go a.Scheduler.Run(ctx).
	Scheduler *syncp.Scheduler

	once    sync.Once
	initErr error
}

// Init loads configuration and wires all dependencies. It is idempotent:
// subsequent calls return the result of the first call without reinitialising.
func (a *App) Init(ctx context.Context) error {
	a.once.Do(func() { a.initErr = a.init(ctx) })
	return a.initErr
}

func (a *App) init(_ context.Context) error {
	cfgPath := a.ConfigPath
	if cfgPath == "" {
		var err error
		cfgPath, err = config.DefaultConfigPath()
		if err != nil {
			return fmt.Errorf("config path: %w", err)
		}
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	a.Config = cfg

	srcs, err := sources.LoadAll(cfg)
	if err != nil {
		return fmt.Errorf("load sources: %w", err)
	}

	storePath := filepath.Join(filepath.Dir(cfgPath), "state.json")
	store, err := jsonfile.Open(storePath)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}

	runner := syncp.New(store, srcs)
	a.Actions = actions.New(store, runner)

	entries := make([]syncp.ScheduleEntry, 0, len(cfg.Sources))
	for _, src := range cfg.Sources {
		entries = append(entries, syncp.ScheduleEntry{
			SourceName: src.Name,
			Interval:   src.SyncIntervalOrDefault(3 * time.Hour),
		})
	}
	auditLog := filepath.Join(filepath.Dir(cfgPath), "sync-audit.jsonl")
	a.Scheduler = syncp.NewScheduler(runner, entries, auditLog)
	return nil
}
