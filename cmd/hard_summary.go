package cmd

import (
	"fmt"

	"github.com/khanalsaroj/gitreport/internal/ai"
	"github.com/khanalsaroj/gitreport/internal/config"
	"github.com/khanalsaroj/gitreport/internal/formatter"
	"github.com/khanalsaroj/gitreport/internal/git"
	"github.com/khanalsaroj/gitreport/internal/summarizer"
	"github.com/khanalsaroj/gitreport/internal/timeutil"
	"github.com/spf13/cobra"
)

var hardSummaryCmd = &cobra.Command{
	Use:   "hard-summary",
	Short: "Generate a deep report from code diffs",
	Long:  `Analyze actual code changes (diffs) and produce a leadership-level engineering report.`,
	RunE:  runHardSummary,
}

func init() {
	addCommonFlags(hardSummaryCmd)
	rootCmd.AddCommand(hardSummaryCmd)
}

func runHardSummary(cmd *cobra.Command, args []string) error {
	if err := validateTimeFlags(); err != nil {
		return err
	}

	since, err := timeutil.Resolve(flagWeek, flagDays, flagMonth)
	if err != nil {
		return fmt.Errorf("resolving time range: %w", err)
	}

	repos, err := git.ResolveRepos(flagProjects)
	if err != nil {
		return fmt.Errorf("resolving repositories: %w", err)
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	provider, err := ai.NewOpenAIProvider()
	if err != nil {
		return fmt.Errorf("initializing AI provider: %w", err)
	}

	diffs, err := git.GetDiffs(repos, since)
	if err != nil {
		return fmt.Errorf("fetching diffs: %w", err)
	}
	if diffs == "" {
		return fmt.Errorf("no diffs found in the specified time range")
	}

	sum := summarizer.NewHardSummary(provider, cfg)

	stream, err := sum.Stream(cmd.Context(), diffs, flagFormat, flagByAuthor)
	if err != nil {
		return fmt.Errorf("generating hard summary: %w", err)
	}

	out := formatter.NewWriter(flagOutput, flagFormat)
	if err := out.Stream(stream); err != nil {
		return fmt.Errorf("writing output: %w", err)
	}

	return nil
}
