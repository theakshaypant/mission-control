// Package cmd defines the devbrief CLI command tree.
package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/theakshaypant/mission-control/internal/app"

	// Register source factories via init().
	_ "github.com/theakshaypant/mission-control/internal/sources/github"
)

// application is the shared dependency container. All subcommands access it
// after PersistentPreRunE has called application.Init.
var application = &app.App{}

var rootCmd = &cobra.Command{
	Use:   "devbrief",
	Short: "Surface what actually needs your attention across your tools.",
	Long: `devbrief is a stateful attention engine for developers.

It syncs items from configured sources, evaluates a set of signals to
decide what actually needs your attention, and presents a focused summary
— filtering out dismissed items, active snoozes, and anything that is
not waiting on you.

Configuration lives in ~/.config/mission-control/config.yaml.

Run 'devbrief help <command>' for detailed usage of any subcommand.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return application.Init(cmd.Context())
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(
		&application.ConfigPath,
		"config", "",
		"path to config file (default: ~/.config/mission-control/config.yaml)",
	)
	rootCmd.AddCommand(syncCmd, summaryCmd, dismissCmd, snoozeCmd)
}

// Execute runs the root command with the given context.
// It prints errors to stderr and returns a non-nil error on failure.
func Execute(ctx context.Context) error {
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "devbrief: %v\n", err)
		return err
	}
	return nil
}
