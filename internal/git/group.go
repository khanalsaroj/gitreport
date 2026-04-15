package git

import (
	"fmt"
	"strings"
	"time"

	"github.com/khanalsaroj/gitreport/internal/timeutil"
)

// GetCommitsByAuthor retrieves commits grouped by author name.
func GetCommitsByAuthor(repos []string, since time.Time) (map[string][]string, error) {
	grouped := make(map[string][]string)
	sinceStr := timeutil.FormatGit(since)

	for _, repo := range repos {
		out, err := runGit(repo, "log",
			"--since="+sinceStr,
			"--pretty=format:%an|%s",
			"--no-merges",
		)
		if err != nil {
			return nil, fmt.Errorf("git log in %q: %w", repo, err)
		}
		if out == "" {
			continue
		}

		repoName := ""
		if len(repos) > 1 {
			repoName = repoBaseName(repo)
		}

		for _, line := range splitLines(out) {
			parts := strings.SplitN(line, "|", 2)
			if len(parts) != 2 {
				continue
			}
			author := strings.TrimSpace(parts[0])
			subject := strings.TrimSpace(parts[1])
			if repoName != "" {
				subject = fmt.Sprintf("[%s] %s", repoName, subject)
			}
			grouped[author] = append(grouped[author], subject)
		}
	}

	return grouped, nil
}
