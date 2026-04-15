package git

import (
	"fmt"
	"strings"
	"time"

	"github.com/khanalsaroj/gitreport/internal/timeutil"
)

// GetDiffs retrieves unified diffs with author info for all commits since the given time across repos.
func GetDiffs(repos []string, since time.Time) (string, error) {
	sinceStr := timeutil.FormatGit(since)
	var parts []string

	for _, repo := range repos {
		// Get commit diffs with author info
		out, err := runGit(repo, "log",
			"--since="+sinceStr,
			"--no-merges",
			"--no-color",
			"--unified=3",
			"--diff-filter=ACMRT",
			"--pretty=format:=== Commit: %H%nAuthor: %an <%ae>%nDate: %ad%nMessage: %s%n",
			"-p",
		)
		if err != nil {
			return "", fmt.Errorf("git log in %q: %w", repo, err)
		}

		if out == "" {
			continue
		}

		if len(repos) > 1 {
			parts = append(parts, fmt.Sprintf("=== Repository: %s ===\n%s", repoBaseName(repo), out))
		} else {
			parts = append(parts, out)
		}
	}

	return strings.Join(parts, "\n\n"), nil
}
