package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var dismissCmd = &cobra.Command{
	Use:   "dismiss <id>",
	Short: "Dismiss an item so it no longer appears in your summary",
	Long: `Permanently suppress an item from your summary.

Dismissed items are never shown in 'devbrief summary', regardless of
whether their signals fire again in future syncs. Dismissal survives
syncs — it is a local decision that overrides source state.

Use 'snooze' instead if you want the item to reappear after a set time.

The item ID is shown in the ID column of 'devbrief summary'.

Example:
  devbrief dismiss github:owner/repo#42`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		if err := application.Actions.DismissItem(cmd.Context(), id); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "dismissed %s\n", id)
		return nil
	},
}
