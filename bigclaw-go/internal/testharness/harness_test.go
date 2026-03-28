package testharness

import (
	"path/filepath"
	"testing"
)

func TestRepoAndProjectRoots(t *testing.T) {
	repoRoot := RepoRoot(t)
	if filepath.Base(repoRoot) != "bigclaw-go" {
		t.Fatalf("expected repo root to end with bigclaw-go, got %q", repoRoot)
	}

	projectRoot := ProjectRoot(t)
	if filepath.Base(projectRoot) != "BIG-GO-923" {
		t.Fatalf("expected project root to end with BIG-GO-923, got %q", projectRoot)
	}
	if got := filepath.Dir(repoRoot); got != projectRoot {
		t.Fatalf("expected project root %q, got %q", got, projectRoot)
	}
}

func TestJoinAndResolveProjectPaths(t *testing.T) {
	if got := JoinRepoRoot(t, "docs", "reports"); got != filepath.Join(RepoRoot(t), "docs", "reports") {
		t.Fatalf("unexpected repo path join: %q", got)
	}
	if got := JoinProjectRoot(t, "src", "bigclaw"); got != filepath.Join(ProjectRoot(t), "src", "bigclaw") {
		t.Fatalf("unexpected project path join: %q", got)
	}
	if got := ResolveProjectPath(t, "bigclaw-go/docs/reports/live-validation-index.json"); got != filepath.Join(RepoRoot(t), "docs", "reports", "live-validation-index.json") {
		t.Fatalf("unexpected resolved path: %q", got)
	}
}
