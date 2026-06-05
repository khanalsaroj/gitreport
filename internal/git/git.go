package git

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// repoBaseName returns the last component of a repo path. It accepts both
// forward- and backslash-separated paths so multi-repo labels are correct on
// Windows as well as Unix.
func repoBaseName(path string) string {
	trimmed := strings.TrimRight(path, `/\`)
	if trimmed == "" {
		return path
	}
	return filepath.Base(trimmed)
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
