package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/khanalsaroj/gitreport/internal/version"
	"github.com/spf13/cobra"
)

var (
	flagWeek     int
	flagDays     int
	flagMonth    int
	flagByAuthor string
	flagProjects string
	flagFormat   string
	flagOutput   string
)

var rootCmd = &cobra.Command{
	Use:   "gitreport",
	Short: "AI-powered Git analytics for engineering reports",
	Long: `gitreport is a developer productivity CLI that transforms raw Git history 
into structured, high-signal engineering reports using AI.

It helps teams and individuals:
  • Summarize weekly development work
  • Track feature delivery and bug fixes
  • Generate leadership-ready reports instantly
  • Understand contribution patterns across teams

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🧠 MODES
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

summary
  Fast analysis using commit messages.
  Best for quick insights and daily/weekly reports.

hard-summary
  Deep analysis using actual code diffs.
  Produces more accurate, context-aware reports.
  Uses AI to understand code changes and team dynamics.
  Recommended for deeper analysis and team reports.
  User more tokens.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
⚙️ TIME FILTERS
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

--week N     Analyze the last N weeks
--days N     Analyze the last N days
--month N    Analyze the last N months

(Default: last 1 week)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✨ OUTPUT OPTIONS
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

--author           Commit by author
--format           text | markdown | json
--output           Save report to file

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🚀 EXAMPLES
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

# Weekly report
gitreport summary --week 1

# Team contribution breakdown
gitreport summary --week 1 --author JohnDoe

# Quick daily snapshot
gitreport summary --days 1

# Deep analysis with markdown export
gitreport hard-summary --week 1 --format markdown --output report.md

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`,
	SilenceUsage:  true,
	SilenceErrors: true,
	Version:       version.String(),
}

func Execute() {
	// Cancel in-flight work (git commands, AI streaming) on Ctrl-C or SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func addCommonFlags(cmd *cobra.Command) {
	cmd.Flags().IntVar(&flagWeek, "week", 0, "Number of weeks to look back")
	cmd.Flags().IntVar(&flagDays, "days", 0, "Number of days to look back")
	cmd.Flags().IntVar(&flagMonth, "month", 0, "Number of months to look back")
	cmd.Flags().StringVar(&flagByAuthor, "author", "", "Commit by author")
	cmd.Flags().StringVar(&flagProjects, "projects", "", "Comma-separated list of repository paths")
	cmd.Flags().StringVar(&flagFormat, "format", "text", "Output format: text | markdown | json")
	cmd.Flags().StringVar(&flagOutput, "output", "", "Output file path (default: stdout)")
}

func validateTimeFlags() error {
	count := 0
	if flagWeek > 0 {
		count++
	}
	if flagDays > 0 {
		count++
	}
	if flagMonth > 0 {
		count++
	}
	if count > 1 {
		return fmt.Errorf("only one time flag allowed: --week, --days, or --month")
	}
	if count == 0 {
		return fmt.Errorf("one time flag is required: --week, --days, or --month")
	}
	return nil
}
