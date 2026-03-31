package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/theakshaypant/mission-control/internal/actions"
)

var summaryJSON bool
var summaryFresh bool

var summaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Show items that need your attention",
	Long: `Display a table of items that currently need your attention.

An item appears in the summary when:
  - At least one attention signal fired during the last sync
  - It has not been dismissed
  - It is not snoozed (or its snooze has expired)

The WHY column shows which signals fired for each item. See your source
documentation for a full list of available signals.

Use --fresh to sync all sources before displaying results, ensuring the
output reflects the latest state from your tools.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if summaryFresh {
			if err := application.Actions.SyncAll(ctx); err != nil {
				return err
			}
		}
		items, err := application.Actions.Summary(ctx)
		if err != nil {
			return err
		}
		if summaryJSON {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(items)
		}
		printSummaryTable(cmd, items)
		return nil
	},
}

func init() {
	summaryCmd.Flags().BoolVar(&summaryJSON, "json", false, "Output as JSON")
	summaryCmd.Flags().BoolVar(&summaryFresh, "fresh", false, "Sync all sources before showing summary")
}

func printSummaryTable(cmd *cobra.Command, items []actions.ItemSummary) {
	if len(items) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "nothing needs your attention")
		return
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTYPE\tTITLE\tUPDATED\tWHY\tURL")
	for _, item := range items {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			item.ID,
			item.Type,
			truncate(item.Title, 60),
			item.UpdatedAt.Local().Format("2006-01-02 15:04"),
			strings.Join(item.ActiveSignals, ", "),
			item.URL,
		)
	}
	_ = w.Flush()
}

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n-1]) + "…"
}
