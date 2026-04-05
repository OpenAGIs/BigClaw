package bootstrap

import (
	"encoding/json"
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

func TestBuildValidationReportReusesSingleCacheAcrossWorkspaces(t *testing.T) {
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
		t.Fatalf("expected 3 workspaces, got %d", report.Summary.WorkspaceCount)
	}
	if !report.Summary.SingleCacheRootReused || !report.Summary.SingleMirrorReused || !report.Summary.SingleSeedReused {
		t.Fatalf("expected single shared cache assets, got %+v", report.Summary)
	}
	if report.Summary.MirrorCreations != 1 || report.Summary.SeedCreations != 1 {
		t.Fatalf("expected single warm-up clone, got %+v", report.Summary)
	}
	if !report.Summary.CloneSuppressedAfterFirst || !report.Summary.CacheReusedAfterFirst || !report.Summary.CleanupPreservedCache {
		t.Fatalf("expected cache reuse summary, got %+v", report.Summary)
	}
	if len(report.BootstrapResults) != 3 || len(report.CleanupResults) != 3 {
		t.Fatalf("unexpected bootstrap/cleanup result counts: %+v", report)
	}
}

func TestWriteValidationReportWritesJSONAndMarkdown(t *testing.T) {
	root := t.TempDir()
	report := ValidationReport{
		RepoURL:          "git@github.com:OpenAGIs/BigClaw.git",
		DefaultBranch:    "main",
		WorkspaceRoot:    filepath.Join(root, "workspaces"),
		IssueIdentifiers: []string{"BIG-1"},
		BootstrapResults: []WorkspaceBootstrapStatus{{
			Workspace:       filepath.Join(root, "workspaces", "BIG-1"),
			CacheRoot:       filepath.Join(root, "repos", "github.com-openagis-bigclaw"),
			CacheKey:        "openagis-bigclaw",
			WorkspaceMode:   "worktree_created",
			CacheReused:     false,
			CloneSuppressed: false,
			MirrorCreated:   true,
			SeedCreated:     true,
		}},
		Summary: ValidationSummary{
			WorkspaceCount:            1,
			SingleCacheRootReused:     true,
			MirrorCreations:           1,
			SeedCreations:             1,
			CloneSuppressedAfterFirst: true,
			CleanupPreservedCache:     true,
		},
	}

	jsonPath, err := WriteValidationReport(report, filepath.Join(root, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatal(err)
	}
	var decoded ValidationReport
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("expected valid JSON report: %v", err)
	}
	if decoded.RepoURL != report.RepoURL || decoded.Summary.WorkspaceCount != report.Summary.WorkspaceCount {
		t.Fatalf("unexpected JSON report contents: %+v", decoded)
	}

	markdownPath, err := WriteValidationReport(report, filepath.Join(root, "report.md"))
	if err != nil {
		t.Fatal(err)
	}
	markdown, err := os.ReadFile(markdownPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(markdown)
	if !strings.Contains(text, "# Symphony bootstrap cache validation") || !strings.Contains(text, "openagis-bigclaw") {
		t.Fatalf("unexpected markdown report contents: %s", text)
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
