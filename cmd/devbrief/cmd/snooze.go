package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	snoozeFor   string
	snoozeUntil string
)

var snoozeCmd = &cobra.Command{
	Use:   "snooze <id>",
	Short: "Snooze an item for a duration or until a specific date/time",
	Long: `Hide an item from your summary until a specified time.

Once the snooze expires the item reappears automatically on the next
'devbrief summary' run, provided its attention signals still fire.

You must provide exactly one of --for or --until:

  --for <duration>   Snooze for a relative duration from now.
                     Supports Go durations (1h30m, 24h) and days (2d, 7d).

  --until <time>     Snooze until an absolute point in time (local timezone).
                     Accepted formats:
                       HH:MM         today at that time  (e.g. 14:30)
                       YYYY-MM-DD    start of that day   (e.g. 2026-04-01)
                       RFC3339       exact timestamp      (e.g. 2026-04-01T09:00:00Z)

Examples:
  devbrief snooze github:owner/repo#42 --for 24h
  devbrief snooze github:owner/repo#42 --for 2d
  devbrief snooze github:owner/repo#42 --until 14:30
  devbrief snooze github:owner/repo#42 --until 2026-04-07`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		until, err := resolveSnoozeTime(snoozeFor, snoozeUntil)
		if err != nil {
			return err
		}
		if err := application.Actions.SnoozeItem(cmd.Context(), id, until); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "snoozed %s until %s\n", id, until.Format(time.RFC3339))
		return nil
	},
}

func init() {
	snoozeCmd.Flags().StringVar(&snoozeFor, "for", "", `duration to snooze (e.g. "2h30m", "24h", "7d")`)
	snoozeCmd.Flags().StringVar(&snoozeUntil, "until", "", `snooze until a time or date (e.g. "14:30", "2026-04-01", RFC3339)`)
	snoozeCmd.MarkFlagsOneRequired("for", "until")
	snoozeCmd.MarkFlagsMutuallyExclusive("for", "until")
}

// resolveSnoozeTime converts CLI flag values into an absolute snooze deadline.
func resolveSnoozeTime(forStr, untilStr string) (time.Time, error) {
	if forStr != "" {
		d, err := parseDuration(forStr)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid --for value %q: %w", forStr, err)
		}
		return time.Now().Add(d), nil
	}
	return parseUntil(untilStr)
}

// parseUntil converts an --until string into an absolute time using local time.
// Accepted formats (in order of precedence):
//
//	RFC3339            "2026-04-01T14:30:00Z"
//	Date only          "2026-04-01"   → start of that day (local)
//	Time of day HH:MM  "14:30"        → today at that time (local)
func parseUntil(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.DateOnly, s); err == nil {
		now := time.Now()
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, now.Location()), nil
	}
	if t, err := time.Parse("15:04", s); err == nil {
		now := time.Now()
		return time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, now.Location()), nil
	}
	return time.Time{}, fmt.Errorf(
		"invalid --until value %q: accepted formats are HH:MM (e.g. 14:30), date (e.g. 2026-04-01), or RFC3339",
		s,
	)
}

// parseDuration extends time.ParseDuration to support "d" (days) as a unit.
func parseDuration(s string) (time.Duration, error) {
	if strings.HasSuffix(s, "d") {
		var days int
		if _, err := fmt.Sscanf(strings.TrimSuffix(s, "d"), "%d", &days); err != nil {
			return 0, fmt.Errorf("invalid day count in %q", s)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}
