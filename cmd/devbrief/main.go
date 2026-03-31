package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/theakshaypant/mission-control/cmd/devbrief/cmd"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := cmd.Execute(ctx); err != nil {
		os.Exit(1)
	}
}
