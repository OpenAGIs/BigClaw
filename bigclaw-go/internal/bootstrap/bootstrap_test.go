package bootstrap

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestCacheRootForRepoUsesRepoSpecificDirectory(t *testing.T) {
	root := t.TempDir()
	cacheRoot := CacheRootForRepo("git@github.com:OpenAGIs/BigClaw.git", filepath.Join(root, "repos"), "")
	want := filepath.Join(root, "repos", "github.com-openagis-bigclaw")
	if cacheRoot != want {
		t.Fatalf("cache root = %s, want %s", cacheRoot, want)
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

func TestSecondWorkspaceReusesWarmCacheWithoutFullClone(t *testing.T) {
	root := t.TempDir()
	remote := initRemoteWithMain(t, root)
	cacheBase := filepath.Join(root, "repos")

	first, err := BootstrapWorkspace(filepath.Join(root, "workspaces", "OPE-322"), "OPE-322", remote, "main", "", cacheBase, "")
	if err != nil {
		t.Fatal(err)
	}
	second, err := BootstrapWorkspace(filepath.Join(root, "workspaces", "OPE-323"), "OPE-323", remote, "main", "", cacheBase, "")
	if err != nil {
		t.Fatal(err)
	}

	if first.CacheRoot != second.CacheRoot {
		t.Fatalf("expected shared cache root, got %s vs %s", first.CacheRoot, second.CacheRoot)
	}
	if !second.CacheReused || !second.CloneSuppressed {
		t.Fatalf("expected warm-cache reuse on second bootstrap, got %+v", second)
	}
	if second.MirrorCreated || second.SeedCreated || second.WorkspaceMode != "worktree_created" {
		t.Fatalf("expected second bootstrap to reuse mirror and seed, got %+v", second)
	}
}

func TestBootstrapWorkspaceReusesExistingIssueWorktree(t *testing.T) {
	root := t.TempDir()
	remote := initRemoteWithMain(t, root)
	cacheBase := filepath.Join(root, "repos")
	workspace := filepath.Join(root, "workspaces", "OPE-324")

	first, err := BootstrapWorkspace(workspace, "OPE-324", remote, "main", "", cacheBase, "")
	if err != nil {
		t.Fatal(err)
	}
	second, err := BootstrapWorkspace(workspace, "OPE-324", remote, "main", "", cacheBase, "")
	if err != nil {
		t.Fatal(err)
	}

	if first.Reused {
		t.Fatalf("expected first bootstrap to create worktree, got %+v", first)
	}
	if !second.Reused || second.WorkspaceMode != "workspace_reused" {
		t.Fatalf("expected existing worktree reuse, got %+v", second)
	}
	if !second.CacheReused || !second.CloneSuppressed || second.Branch != "symphony/OPE-324" {
		t.Fatalf("unexpected reuse status: %+v", second)
	}
}

func TestCleanupWorkspacePreservesSharedCacheForFutureReuse(t *testing.T) {
	root := t.TempDir()
	remote := initRemoteWithMain(t, root)
	cacheBase := filepath.Join(root, "repos")
	cacheRoot := CacheRootForRepo(remote, cacheBase, "")
	workspace := filepath.Join(root, "workspaces", "OPE-325")

	if _, err := BootstrapWorkspace(workspace, "OPE-325", remote, "main", "", cacheBase, ""); err != nil {
		t.Fatal(err)
	}
	status, err := CleanupWorkspace(workspace, "OPE-325", remote, "main", "", cacheBase, "")
	if err != nil {
		t.Fatal(err)
	}
	followUp, err := BootstrapWorkspace(filepath.Join(root, "workspaces", "OPE-326"), "OPE-326", remote, "main", "", cacheBase, "")
	if err != nil {
		t.Fatal(err)
	}

	if !status.Removed || pathExists(workspace) {
		t.Fatalf("expected workspace cleanup to remove worktree, got %+v", status)
	}
	if !pathExists(filepath.Join(cacheRoot, "mirror.git", "HEAD")) || !pathExists(filepath.Join(cacheRoot, "seed", ".git")) {
		t.Fatalf("expected cache assets to survive cleanup under %s", cacheRoot)
	}
	if !followUp.CacheReused || !followUp.CloneSuppressed || followUp.MirrorCreated || followUp.SeedCreated {
		t.Fatalf("expected follow-up bootstrap to reuse cache, got %+v", followUp)
	}
}

func TestBootstrapRecoversFromStaleSeedDirectoryWithoutRemoteReclone(t *testing.T) {
	root := t.TempDir()
	remote := initRemoteWithMain(t, root)
	cacheBase := filepath.Join(root, "repos")

	first, err := BootstrapWorkspace(filepath.Join(root, "workspaces", "OPE-327"), "OPE-327", remote, "main", "", cacheBase, "")
	if err != nil {
		t.Fatal(err)
	}
	cacheRoot := first.CacheRoot

	if _, err := CleanupWorkspace(filepath.Join(root, "workspaces", "OPE-327"), "OPE-327", remote, "main", "", cacheBase, ""); err != nil {
		t.Fatal(err)
	}
	seedPath := filepath.Join(cacheRoot, "seed")
	if err := os.RemoveAll(seedPath); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(seedPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(seedPath, "stale.txt"), []byte("stale\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	recovered, err := BootstrapWorkspace(filepath.Join(root, "workspaces", "OPE-328"), "OPE-328", remote, "main", "", cacheBase, "")
	if err != nil {
		t.Fatal(err)
	}

	if recovered.CacheReused || !recovered.CloneSuppressed || recovered.MirrorCreated || !recovered.SeedCreated {
		t.Fatalf("expected stale seed recovery without remote reclone, got %+v", recovered)
	}
	if !pathExists(filepath.Join(cacheRoot, "mirror.git", "HEAD")) || !pathExists(filepath.Join(cacheRoot, "seed", ".git")) {
		t.Fatalf("expected mirror and recovered seed to exist under %s", cacheRoot)
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

func TestValidationReportCoversThreeWorkspacesWithOneCache(t *testing.T) {
	root := t.TempDir()
	remote := initRemoteWithMain(t, root)
	report, err := BuildValidationReport(
		remote,
		filepath.Join(root, "validation-workspaces"),
		[]string{"OPE-272", "OPE-273", "OPE-274"},
		"main",
		"",
		filepath.Join(root, "repos"),
		"",
		true,
	)
	if err != nil {
		t.Fatal(err)
	}

	if report.Summary.WorkspaceCount != 3 {
		t.Fatalf("workspace count = %d, want 3", report.Summary.WorkspaceCount)
	}
	if !report.Summary.SingleCacheRootReused || !report.Summary.SingleMirrorReused || !report.Summary.SingleSeedReused {
		t.Fatalf("expected single cache/mirror/seed reuse, got %+v", report.Summary)
	}
	if report.Summary.MirrorCreations != 1 || report.Summary.SeedCreations != 1 {
		t.Fatalf("expected one mirror and one seed creation, got %+v", report.Summary)
	}
	if !report.Summary.CloneSuppressedAfterFirst || !report.Summary.CacheReusedAfterFirst || !report.Summary.CleanupPreservedCache {
		t.Fatalf("expected reuse and cleanup summary flags, got %+v", report.Summary)
	}
}

func TestWithCacheLockSerializesAcrossProcesses(t *testing.T) {
	if os.Getenv("BOOTSTRAP_LOCK_HELPER") == "1" {
		lockRoot := os.Getenv("BOOTSTRAP_LOCK_ROOT")
		releaseFile := os.Getenv("BOOTSTRAP_LOCK_RELEASE")
		heldFile := os.Getenv("BOOTSTRAP_LOCK_HELD")
		if err := withCacheLock(lockRoot, func() error {
			if err := os.WriteFile(heldFile, []byte("held"), 0o644); err != nil {
				return err
			}
			for {
				if _, err := os.Stat(releaseFile); err == nil {
					return nil
				}
				time.Sleep(25 * time.Millisecond)
			}
		}); err != nil {
			t.Fatal(err)
		}
		return
	}

	root := t.TempDir()
	lockRoot := filepath.Join(root, "cache")
	releaseFile := filepath.Join(root, "release")
	heldFile := filepath.Join(root, "held")

	holder := exec.Command(os.Args[0], "-test.run=TestWithCacheLockSerializesAcrossProcesses")
	holder.Env = append(
		os.Environ(),
		"BOOTSTRAP_LOCK_HELPER=1",
		"BOOTSTRAP_LOCK_ROOT="+lockRoot,
		"BOOTSTRAP_LOCK_RELEASE="+releaseFile,
		"BOOTSTRAP_LOCK_HELD="+heldFile,
	)
	if err := holder.Start(); err != nil {
		t.Fatalf("failed starting lock holder: %v", err)
	}
	defer holder.Process.Kill()

	deadline := time.Now().Add(3 * time.Second)
	for {
		if _, err := os.Stat(heldFile); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for holder to acquire cache lock")
		}
		time.Sleep(25 * time.Millisecond)
	}

	waitStartedAt := time.Now()
	done := make(chan error, 1)
	go func() {
		done <- withCacheLock(lockRoot, func() error { return nil })
	}()

	select {
	case err := <-done:
		t.Fatalf("lock should block while holder is active, got %v", err)
	case <-time.After(150 * time.Millisecond):
	}

	if err := os.WriteFile(releaseFile, []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for cache lock release")
	}

	if err := holder.Wait(); err != nil {
		t.Fatalf("holder process failed: %v", err)
	}

	if time.Since(waitStartedAt) < 150*time.Millisecond {
		t.Fatalf("expected cache lock acquisition to wait for holder")
	}
}

func TestBootstrapHelpersUniqueJoinAndPathExists(t *testing.T) {
	if got := uniqueStrings([]string{"alpha", "beta", "alpha", "", "beta", "gamma"}); strings.Join(got, ",") != "alpha,beta,,gamma" {
		t.Fatalf("expected uniqueStrings to preserve first-seen order, got %q", strings.Join(got, ","))
	}

	if got := join(nil); got != "" {
		t.Fatalf("expected empty join result, got %q", got)
	}
	if got := join([]string{"git", "status", "--short"}); got != "git status --short" {
		t.Fatalf("unexpected join result %q", got)
	}
	if got := joinLines(nil); got != "" {
		t.Fatalf("expected empty joinLines result, got %q", got)
	}
	if got := joinLines([]string{"alpha", "beta", "gamma"}); got != "alpha\nbeta\ngamma" {
		t.Fatalf("unexpected joinLines result %q", got)
	}

	root := t.TempDir()
	existing := filepath.Join(root, "exists.txt")
	if err := os.WriteFile(existing, []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !pathExists(existing) {
		t.Fatalf("expected existing path to be detected")
	}
	if pathExists(filepath.Join(root, "missing.txt")) {
		t.Fatalf("expected missing path to stay false")
	}
}
