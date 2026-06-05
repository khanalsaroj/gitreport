package cmd

import (
	"fmt"
	"strings"

	"github.com/khanalsaroj/gitreport/internal/ai"
	"github.com/khanalsaroj/gitreport/internal/config"
	"github.com/khanalsaroj/gitreport/internal/formatter"
	"github.com/khanalsaroj/gitreport/internal/git"
	"github.com/khanalsaroj/gitreport/internal/summarizer"
	"github.com/khanalsaroj/gitreport/internal/timeutil"
	"github.com/spf13/cobra"
)

var summaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Generate a report from commit messages",
	Long:  `Analyze git commit messages and produce a structured engineering report.`,
	RunE:  runSummary,
}

func init() {
	addCommonFlags(summaryCmd)
	rootCmd.AddCommand(summaryCmd)
}

func runSummary(cmd *cobra.Command, args []string) error {
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

	sum := summarizer.NewSummary(provider, cfg)

	var commitData string

	grouped, err := git.GetCommitsByAuthor(repos, since)
	if err != nil {
		return fmt.Errorf("fetching commits by author: %w", err)
	}
	if len(grouped) == 0 {
		return fmt.Errorf("no commits found in the specified time range")
	}

	if flagByAuthor == "" {
		commitData = formatGroupedCommits(grouped)
	} else {
		var filtered []string
		for author, commits := range grouped {
			if strings.Contains(strings.ToLower(author), strings.ToLower(flagByAuthor)) {
				filtered = append(filtered, commits...)
			}
		}

		if len(filtered) == 0 {
			var authors []string
			for a := range grouped {
				authors = append(authors, a)
			}
			return fmt.Errorf(
				"no commits found for author: %s. available authors: %s",
				flagByAuthor,
				strings.Join(authors, ", "),
			)
		}

		commitData = strings.Join(filtered, "\n")
	}

	stream, err := sum.Stream(cmd.Context(), commitData, flagFormat, flagByAuthor)
	if err != nil {
		return fmt.Errorf("generating summary: %w", err)
	}

	out := formatter.NewWriter(flagOutput, flagFormat)
	if err := out.Stream(stream); err != nil {
		return fmt.Errorf("writing output: %w", err)
	}

	return nil
}

func formatGroupedCommits(grouped map[string][]string) string {
	var sb strings.Builder
	for author, commits := range grouped {
		sb.WriteString(fmt.Sprintf("Author: %s\n", author))
		for _, c := range commits {
			sb.WriteString(fmt.Sprintf("  - %s\n", c))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}
