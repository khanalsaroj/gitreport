package cmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/khanalsaroj/gitreport/internal/ai"
	"github.com/spf13/cobra"
)

var providersCmd = &cobra.Command{
	Use:   "providers",
	Short: "List AI providers and their availability",
	Long: `Show every known AI provider in priority order, whether it is currently
available, and which one would be used. Use this to verify detection and
diagnose why a provider is being skipped.`,
	Args: cobra.NoArgs,
	RunE: runProviders,
}

func init() {
	rootCmd.AddCommand(providersCmd)
}

func runProviders(cmd *cobra.Command, args []string) error {
	statuses, err := ai.Statuses(cmd.Context())
	if err != nil {
		return fmt.Errorf("inspecting providers: %w", err)
	}

	out := cmd.OutOrStdout()
	tw := tabwriter.NewWriter(out, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "PROVIDER\tNAME\tSTATUS\tDETAIL")
	for _, s := range statuses {
		status := "unavailable"
		if s.Available {
			status = "available"
			if s.Primary {
				status = "available (primary)"
			}
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", s.ID, s.DisplayName, status, s.Detail)
	}
	if err := tw.Flush(); err != nil {
		return err
	}

	fmt.Fprintln(out, "\nPriority is configurable via setting.json (\"priority\") or --provider.")
	return nil
}
