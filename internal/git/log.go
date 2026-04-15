package git

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/khanalsaroj/gitreport/internal/timeutil"
)

// GetCommitMessages retrieves commit message subjects from all repos since the given time.
func GetCommitMessages(repos []string, since time.Time) ([]string, error) {
	var all []string
	sinceStr := timeutil.FormatGit(since)

	for _, repo := range repos {
		out, err := runGit(repo, "log",
			"--since="+sinceStr,
			"--pretty=format:%an: %s",
			"--no-merges",
		)
		if err != nil {
			return nil, fmt.Errorf("git log in %q: %w", repo, err)
		}
		if out == "" {
			continue
		}

		lines := splitLines(out)

		if len(repos) > 1 {
			repoName := repoBaseName(repo)
			for i, l := range lines {
				lines[i] = fmt.Sprintf("[%s] %s", repoName, l)
			}
		}

		all = append(all, lines...)
	}

	return all, nil
}

// repoBaseName returns the last component of a repo path.
func repoBaseName(path string) string {
	parts := strings.Split(strings.TrimRight(path, "/"), "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return path
}

// splitLines splits output into non-empty lines.
func splitLines(s string) []string {
	var result []string
	for _, l := range strings.Split(s, "\n") {
		l = strings.TrimSpace(l)
		if l != "" {
			result = append(result, l)
		}
	}
	return result
}

// runGit executes a git command in the given directory and returns stdout.
func runGit(dir string, args ...string) (string, error) {
	cmdArgs := append([]string{"-C", dir}, args...)
	cmd := exec.Command("git", cmdArgs...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("%w: %s", err, string(exitErr.Stderr))
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
