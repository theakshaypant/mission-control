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

	"gopkg.in/yaml.v3"

	"github.com/theakshaypant/mission-control/internal/actions"
	"github.com/theakshaypant/mission-control/internal/config"
	"github.com/theakshaypant/mission-control/internal/core"
	"github.com/theakshaypant/mission-control/internal/sources"
	"github.com/theakshaypant/mission-control/internal/store/jsonfile"
	syncp "github.com/theakshaypant/mission-control/internal/sync"

	// Register source factories via init().
	_ "github.com/theakshaypant/mission-control/internal/sources/github"
	_ "github.com/theakshaypant/mission-control/internal/sources/jira"
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

	once    sync.Once
	initErr error

	// hot-reload state — guarded by mu.
	mu          sync.Mutex
	appCtx      context.Context
	schedCancel context.CancelFunc
	scheduler   *syncp.Scheduler
	store       core.Store
	auditLog    string
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
	// Save the resolved path so Reload can find the file.
	a.ConfigPath = cfgPath

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
	a.store = store

	runner := syncp.New(store, srcs)
	a.Actions = actions.New(store, runner)

	entries := make([]syncp.ScheduleEntry, 0, len(cfg.Sources))
	for _, src := range cfg.Sources {
		entries = append(entries, syncp.ScheduleEntry{
			SourceName: src.Name,
			Interval:   src.SyncIntervalOrDefault(3 * time.Hour),
		})
	}
	a.auditLog = filepath.Join(filepath.Dir(cfgPath), "sync-audit.jsonl")
	a.scheduler = syncp.NewScheduler(runner, entries, a.auditLog)
	return nil
}

// StartScheduler launches background syncs and returns immediately.
// Call this once after Init, before starting the HTTP server.
func (a *App) StartScheduler(appCtx context.Context) {
	a.mu.Lock()
	a.appCtx = appCtx
	ctx, cancel := context.WithCancel(appCtx)
	a.schedCancel = cancel
	sched := a.scheduler
	a.mu.Unlock()
	go sched.Run(ctx)
}

// GetSourcesYAML reads the config file from disk and returns just the sources
// section as YAML text. Reading from disk means the editor always reflects the
// file's actual state, including any manual edits made outside the dashboard.
func (a *App) GetSourcesYAML() (string, error) {
	cfg, err := config.Load(a.ConfigPath)
	if err != nil {
		return "", fmt.Errorf("read config: %w", err)
	}
	data, err := yaml.Marshal(cfg.Sources)
	if err != nil {
		return "", fmt.Errorf("marshal sources: %w", err)
	}
	return string(data), nil
}

// ReloadFromYAML validates newYAML as a YAML sources list, dismisses items
// from any removed sources, saves the new config to disk, and hot-swaps the
// runner and scheduler without restarting the server.
func (a *App) ReloadFromYAML(ctx context.Context, newYAML string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Parse and validate. Accept either a full config file (with a "sources:"
	// key) or a bare sources list — both are valid inputs from the editor.
	newSources, err := parseSources([]byte(newYAML))
	if err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}
	newCfg := &config.AppConfig{Sources: newSources, Server: a.Config.Server}
	srcs, err := sources.LoadAll(newCfg)
	if err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// Dismiss items from sources that no longer exist in the new config.
	newNames := make(map[string]bool, len(newSources))
	for _, s := range newSources {
		newNames[s.Name] = true
	}
	for _, s := range a.Config.Sources {
		if !newNames[s.Name] {
			if err := a.Actions.DismissSource(ctx, s.Name); err != nil {
				return fmt.Errorf("dismiss removed source %q: %w", s.Name, err)
			}
		}
	}

	// Reset the sync cursor for any source whose config changed. This forces a
	// full fetch on the next sync so items from added repos are not skipped by
	// the incremental cursor.
	oldByName := make(map[string]config.RawSourceConfig, len(a.Config.Sources))
	for _, s := range a.Config.Sources {
		oldByName[s.Name] = s
	}
	for _, newSrc := range newSources {
		oldSrc, existed := oldByName[newSrc.Name]
		if !existed {
			continue // brand-new source name — no cursor exists yet
		}
		oldData, _ := yaml.Marshal(oldSrc)
		newData, _ := yaml.Marshal(newSrc)
		if string(oldData) != string(newData) {
			_ = a.store.SetLastSyncedAt(ctx, newSrc.Name, time.Time{})
		}
	}

	// Save to disk.
	if err := config.Save(newCfg, a.ConfigPath); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	// Stop the current scheduler.
	if a.schedCancel != nil {
		a.schedCancel()
	}

	// Swap in a new runner and scheduler.
	runner := syncp.New(a.store, srcs)
	a.Actions.SetRunner(runner)

	entries := make([]syncp.ScheduleEntry, 0, len(newSources))
	for _, src := range newSources {
		entries = append(entries, syncp.ScheduleEntry{
			SourceName: src.Name,
			Interval:   src.SyncIntervalOrDefault(3 * time.Hour),
		})
	}
	a.scheduler = syncp.NewScheduler(runner, entries, a.auditLog)

	schedCtx, cancel := context.WithCancel(a.appCtx)
	a.schedCancel = cancel
	go a.scheduler.Run(schedCtx)

	a.Config = newCfg
	return nil
}

// parseSources unmarshals YAML that is either a bare sources list or a full
// config file containing a "sources:" key. Both are valid inputs for the
// dashboard config editor so users can paste either format without error.
func parseSources(data []byte) ([]config.RawSourceConfig, error) {
	// Try as a full config first (map with "sources:" key).
	var full config.AppConfig
	if err := yaml.Unmarshal(data, &full); err == nil && len(full.Sources) > 0 {
		return full.Sources, nil
	}
	// Fall back to a bare list of sources.
	var sources []config.RawSourceConfig
	if err := yaml.Unmarshal(data, &sources); err != nil {
		return nil, err
	}
	return sources, nil
}
