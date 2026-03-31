package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/theakshaypant/mission-control/internal/config"
	"github.com/theakshaypant/mission-control/internal/sources"
	syncp "github.com/theakshaypant/mission-control/internal/sync"
	"github.com/theakshaypant/mission-control/internal/store/jsonfile"

	// Register source factories via init().
	_ "github.com/theakshaypant/mission-control/internal/sources/github"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "devbrief: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()

	cfgPath, err := config.DefaultConfigPath()
	if err != nil {
		return fmt.Errorf("config path: %w", err)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

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
	if err := runner.SyncAll(ctx); err != nil {
		return fmt.Errorf("sync: %w", err)
	}

	fmt.Println("sync complete")
	return nil
}
