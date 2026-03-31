package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync [source]",
	Short: "Sync items from all sources, or a single named source",
	Long: `Fetch the latest items from configured sources and update local state.

Without arguments, all configured sources are synced in sequence.
Pass a source name to sync only that source.

Example:
  devbrief sync
  devbrief sync work-github

Source names come from the 'name' field in your config file. If a source
has no name, its type is used as the name.

Note: if the background server is running, it syncs sources automatically
on a configurable interval. Use this command to force an immediate refresh
or to sync a single source during debugging.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if len(args) == 1 {
			if err := application.Actions.SyncSource(ctx, args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "synced %s\n", args[0])
			return nil
		}
		if err := application.Actions.SyncAll(ctx); err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "sync complete")
		return nil
	},
}
