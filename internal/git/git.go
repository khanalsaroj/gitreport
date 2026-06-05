package git

import (
	"fmt"
	"os/exec"
	"strings"
)

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
