package bootstrap

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func gitOut(t *testing.T, repo string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = repo
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v failed: %v", args, err)
	}
	return trim(string(output))
}

func initBootstrapRepo(t *testing.T, repo string, branch string) {
	t.Helper()
	for _, args := range [][]string{
		{"init", "-b", branch},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "Test User"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v (%s)", args, err, string(output))
		}
	}
}

func commitBootstrapFile(t *testing.T, repo string, name string, content string, message string) string {
	t.Helper()
	if err := os.WriteFile(filepath.Join(repo, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", name}, {"commit", "-m", message}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v (%s)", args, err, string(output))
		}
	}
	return gitOut(t, repo, "rev-parse", "HEAD")
}

func initRemoteWithMain(t *testing.T, root string) string {
	t.Helper()
	remote := filepath.Join(root, "remote.git")
	if output, err := exec.Command("git", "init", "--bare", "--initial-branch=main", remote).CombinedOutput(); err != nil {
		t.Fatalf("git init --bare failed: %v (%s)", err, string(output))
	}
	source := filepath.Join(root, "source")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	initBootstrapRepo(t, source, "main")
	cmd := exec.Command("git", "remote", "add", "origin", remote)
	cmd.Dir = source
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git remote add failed: %v (%s)", err, string(output))
	}
	commitBootstrapFile(t, source, "README.md", "hello\n", "initial")
	cmd = exec.Command("git", "push", "-u", "origin", "main")
	cmd.Dir = source
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git push failed: %v (%s)", err, string(output))
	}
	return remote
}

func TestRepoCacheKeyDerivesFromRepoLocator(t *testing.T) {
	if got := RepoCacheKey("git@github.com:OpenAGIs/BigClaw.git", ""); got != "github.com-openagis-bigclaw" {
		t.Fatalf("unexpected cache key %s", got)
	}
	if got := RepoCacheKey("https://github.com/OpenAGIs/BigClaw.git", ""); got != "github.com-openagis-bigclaw" {
		t.Fatalf("unexpected cache key %s", got)
	}
	if got := RepoCacheKey("git@github.com:OpenAGIs/BigClaw.git", "Team/BigClaw"); got != "team-bigclaw" {
		t.Fatalf("unexpected overridden cache key %s", got)
	}
}

func TestBootstrapWorkspaceCreatesSharedWorktreeFromLocalSeed(t *testing.T) {
	root := t.TempDir()
	remote := initRemoteWithMain(t, root)
	cacheBase := filepath.Join(root, "repos")
	workspace := filepath.Join(root, "workspaces", "OPE-321")

	status, err := BootstrapWorkspace(workspace, "OPE-321", remote, "main", "", cacheBase, "")
	if err != nil {
		t.Fatal(err)
	}

	expectedCacheRoot := CacheRootForRepo(remote, cacheBase, "")
	if status.Reused {
		t.Fatalf("expected fresh bootstrap, got reused")
	}
	if status.Branch != "symphony/OPE-321" {
		t.Fatalf("unexpected branch %s", status.Branch)
	}
	if status.WorkspaceMode != "worktree_created" {
		t.Fatalf("unexpected workspace mode %s", status.WorkspaceMode)
	}
	if status.CacheRoot != expectedCacheRoot {
		t.Fatalf("expected cache root %s, got %s", expectedCacheRoot, status.CacheRoot)
	}
	if !pathExists(filepath.Join(expectedCacheRoot, "mirror.git", "HEAD")) || !pathExists(filepath.Join(expectedCacheRoot, "seed", ".git")) {
		t.Fatalf("expected warm cache assets under %s", expectedCacheRoot)
	}
	if !pathExists(filepath.Join(workspace, ".git")) {
		t.Fatalf("expected worktree git dir at %s", workspace)
	}
	if got := gitOut(t, workspace, "branch", "--show-current"); got != "symphony/OPE-321" {
		t.Fatalf("unexpected workspace branch %s", got)
	}
	if body, err := os.ReadFile(filepath.Join(workspace, "README.md")); err != nil || string(body) != "hello\n" {
		t.Fatalf("unexpected workspace README: %v %q", err, string(body))
	}
}

func TestCleanupWorkspacePrunesWorktreeAndBootstrapBranch(t *testing.T) {
	root := t.TempDir()
	remote := initRemoteWithMain(t, root)
	cacheBase := filepath.Join(root, "repos")
	workspace := filepath.Join(root, "workspaces", "OPE-329")
	cacheRoot := CacheRootForRepo(remote, cacheBase, "")

	if _, err := BootstrapWorkspace(workspace, "OPE-329", remote, "main", "", cacheBase, ""); err != nil {
		t.Fatal(err)
	}
	status, err := CleanupWorkspace(workspace, "OPE-329", remote, "main", "", cacheBase, "")
	if err != nil {
		t.Fatal(err)
	}
	if !status.Removed {
		t.Fatalf("expected removed worktree, got %+v", status)
	}
	if pathExists(workspace) {
		t.Fatalf("expected workspace to be removed")
	}
	branches := splitLines(gitOut(t, filepath.Join(cacheRoot, "seed"), "branch", "--format", "%(refname:short)"))
	for _, branch := range branches {
		if branch == "symphony/OPE-329" {
			t.Fatalf("unexpected lingering bootstrap branch")
		}
	}
	worktreeList := gitOut(t, filepath.Join(cacheRoot, "seed"), "worktree", "list", "--porcelain")
	if strings.Contains(worktreeList, filepath.Clean(workspace)) {
		t.Fatalf("unexpected lingering worktree registration")
	}
}
