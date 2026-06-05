package git

import (
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestRepoBaseName(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/srv/api", "api"},
		{"/srv/api/", "api"},
		{"repo", "repo"},
		{"/a/b/c/frontend", "frontend"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := repoBaseName(tt.path); got != tt.want {
			t.Errorf("repoBaseName(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestSplitLines(t *testing.T) {
	in := "  one  \n\n two\n\t\nthree\n"
	got := splitLines(in)
	want := []string{"one", "two", "three"}
	if len(got) != len(want) {
		t.Fatalf("splitLines() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, got[i], want[i])
		}
	}
}

// --- Integration tests against a real git repo -----------------------------

func gitAvailable(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available in PATH")
	}
}

// initRepo creates a temporary git repo with one commit and returns its path.
func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(cmd.Environ(),
			"GIT_AUTHOR_NAME=Test Author", "GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=Test Author", "GIT_COMMITTER_EMAIL=test@example.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	run("init")
	run("config", "user.name", "Test Author")
	run("config", "user.email", "test@example.com")
	run("commit", "--allow-empty", "-m", "feat: add login flow")
	return dir
}

func TestResolveReposValidAndInvalid(t *testing.T) {
	gitAvailable(t)
	repo := initRepo(t)

	repos, err := ResolveRepos(repo)
	if err != nil {
		t.Fatalf("ResolveRepos(valid) error: %v", err)
	}
	if len(repos) != 1 || repos[0] != repo {
		t.Errorf("ResolveRepos() = %v, want [%s]", repos, repo)
	}

	if _, err := ResolveRepos(t.TempDir()); err == nil {
		t.Error("ResolveRepos(non-repo) expected error, got nil")
	}
}

func TestGetCommitsByAuthor(t *testing.T) {
	gitAvailable(t)
	repo := initRepo(t)

	grouped, err := GetCommitsByAuthor([]string{repo}, time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatalf("GetCommitsByAuthor error: %v", err)
	}
	commits, ok := grouped["Test Author"]
	if !ok {
		t.Fatalf("expected author 'Test Author' in %v", grouped)
	}
	if len(commits) != 1 || !strings.Contains(commits[0], "add login flow") {
		t.Errorf("commits = %v, want the login flow subject", commits)
	}
}

func TestGetCommitsByAuthorMultiRepoLabels(t *testing.T) {
	gitAvailable(t)
	repoA := initRepo(t)
	repoB := initRepo(t)

	grouped, err := GetCommitsByAuthor([]string{repoA, repoB}, time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatalf("GetCommitsByAuthor error: %v", err)
	}
	// With more than one repo, each subject is prefixed with the repo base name.
	for _, c := range grouped["Test Author"] {
		if !strings.HasPrefix(c, "[") {
			t.Errorf("multi-repo commit not labelled with repo name: %q", c)
		}
	}
}
