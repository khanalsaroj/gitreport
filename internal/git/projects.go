package git

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ResolveRepos determines the list of git repository paths based on flags.
func ResolveRepos(projects string) ([]string, error) {
	if projects != "" {
		paths := strings.Split(projects, ",")
		var repos []string
		for _, p := range paths {
			p = strings.TrimSpace(p)
			if err := validateRepo(p); err != nil {
				return nil, fmt.Errorf("invalid repo %q: %w", p, err)
			}
			repos = append(repos, p)
		}
		return repos, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting current directory: %w", err)
	}
	if err := validateRepo(cwd); err != nil {
		return nil, fmt.Errorf("current directory is not a git repository: %w", err)
	}
	return []string{cwd}, nil
}

// validateRepo checks that a path contains a git repository.
func validateRepo(path string) error {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--git-dir")
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("not a git repository")
	}
	return nil
}
