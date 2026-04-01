package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/theakshaypant/mission-control/internal/api"
	"github.com/theakshaypant/mission-control/internal/app"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "server: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		configPath string
		addr       string
	)
	flag.StringVar(&configPath, "config", "", "path to config file (default: ~/.config/mission-control/config.yaml)")
	flag.StringVar(&addr, "addr", "", "address to listen on (overrides config server.addr, default :5040)")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	a := &app.App{ConfigPath: configPath}
	if err := a.Init(ctx); err != nil {
		return err
	}

	if addr == "" {
		addr = a.Config.ServerAddr()
	}

	a.StartScheduler(ctx)

	fmt.Fprintf(os.Stderr, "server: listening on %s\n", addr)
	return api.New(addr, a.Actions, uiFiles(), a.GetSourcesYAML, a.ReloadFromYAML).ListenAndServe(ctx)
}
