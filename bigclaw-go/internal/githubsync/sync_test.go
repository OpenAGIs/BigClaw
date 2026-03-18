package githubsync

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func gitOutput(t *testing.T, repo string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = repo
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v failed: %v", args, err)
	}
	return stringTrimSpace(string(output))
}

func initRepo(t *testing.T, repo string) {
	t.Helper()
	cmds := [][]string{
		{"init"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "Test User"},
	}
	for _, args := range cmds {
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v (%s)", args, err, string(output))
		}
	}
}

func commitFile(t *testing.T, repo string, name string, content string, message string) string {
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
	return gitOutput(t, repo, "rev-parse", "HEAD")
}

func TestInstallGitHooksConfiguresCoreHooksPath(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	initRepo(t, repo)
	hooksDir := filepath.Join(repo, ".githooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	hookPath := filepath.Join(hooksDir, "post-commit")
	if err := os.WriteFile(hookPath, []byte("#!/usr/bin/env bash\nexit 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	installed, err := InstallGitHooks(repo, ".githooks")
	if err != nil {
		t.Fatal(err)
	}

	if installed != hooksDir {
		t.Fatalf("expected hooks dir %s, got %s", hooksDir, installed)
	}
	if got := gitOutput(t, repo, "config", "--get", "core.hooksPath"); got != ".githooks" {
		t.Fatalf("expected .githooks, got %s", got)
	}
	info, err := os.Stat(hookPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&0o111 == 0 {
		t.Fatalf("expected executable hook bits for %s", hookPath)
	}
}

func TestEnsureRepoSyncPushesHeadToOrigin(t *testing.T) {
	tmp := t.TempDir()
	remote := filepath.Join(tmp, "remote.git")
	if output, err := exec.Command("git", "init", "--bare", remote).CombinedOutput(); err != nil {
		t.Fatalf("git init --bare failed: %v (%s)", err, string(output))
	}

	repo := filepath.Join(tmp, "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	initRepo(t, repo)
	cmd := exec.Command("git", "remote", "add", "origin", remote)
	cmd.Dir = repo
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git remote add failed: %v (%s)", err, string(output))
	}
	localSHA := commitFile(t, repo, "README.md", "hello\n", "initial commit")

	status, err := EnsureRepoSync(repo, "origin", true, false)
	if err != nil {
		t.Fatal(err)
	}

	if !status.Pushed || !status.Synced {
		t.Fatalf("expected pushed synced status, got %+v", status)
	}
	if status.LocalSHA != localSHA || status.RemoteSHA != localSHA {
		t.Fatalf("expected both SHAs to equal %s, got %+v", localSHA, status)
	}
}

func TestEnsureRepoSyncPublishesMissingIssueBranchEvenWhenHeadMatchesRemoteDefault(t *testing.T) {
	tmp := t.TempDir()
	remote := filepath.Join(tmp, "remote.git")
	if output, err := exec.Command("git", "init", "--bare", "--initial-branch=main", remote).CombinedOutput(); err != nil {
		t.Fatalf("git init --bare failed: %v (%s)", err, string(output))
	}

	source := filepath.Join(tmp, "source")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	initRepo(t, source)
	if output, err := exec.Command("git", "-C", source, "branch", "-M", "main").CombinedOutput(); err != nil {
		t.Fatalf("git branch -M failed: %v (%s)", err, string(output))
	}
	cmd := exec.Command("git", "remote", "add", "origin", remote)
	cmd.Dir = source
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git remote add failed: %v (%s)", err, string(output))
	}
	localSHA := commitFile(t, source, "README.md", "hello\n", "initial commit")
	cmd = exec.Command("git", "push", "-u", "origin", "main")
	cmd.Dir = source
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git push main failed: %v (%s)", err, string(output))
	}

	workspace := filepath.Join(tmp, "workspace")
	if output, err := exec.Command("git", "clone", remote, workspace).CombinedOutput(); err != nil {
		t.Fatalf("git clone failed: %v (%s)", err, string(output))
	}
	if output, err := exec.Command("git", "-C", workspace, "checkout", "-b", "symphony/BIG-GOM-307").CombinedOutput(); err != nil {
		t.Fatalf("git checkout -b failed: %v (%s)", err, string(output))
	}

	status, err := InspectRepoSync(workspace, "origin")
	if err != nil {
		t.Fatal(err)
	}
	if status.RemoteExists {
		t.Fatalf("expected issue branch to be missing before sync, got %+v", status)
	}
	if !status.Synced {
		t.Fatalf("expected pre-push status to detect matching default branch SHA, got %+v", status)
	}

	status, err = EnsureRepoSync(workspace, "origin", true, false)
	if err != nil {
		t.Fatal(err)
	}
	if !status.Pushed || !status.RemoteExists || !status.Synced {
		t.Fatalf("expected missing issue branch to be published and synced, got %+v", status)
	}
	if status.LocalSHA != localSHA || status.RemoteSHA != localSHA {
		t.Fatalf("expected published branch SHA %s, got %+v", localSHA, status)
	}
}

func TestEnsureRepoSyncFastForwardsCleanBranchBeforePush(t *testing.T) {
	tmp := t.TempDir()
	remote := filepath.Join(tmp, "remote.git")
	if output, err := exec.Command("git", "init", "--bare", "--initial-branch=main", remote).CombinedOutput(); err != nil {
		t.Fatalf("git init --bare failed: %v (%s)", err, string(output))
	}

	seed := filepath.Join(tmp, "seed")
	if err := os.MkdirAll(seed, 0o755); err != nil {
		t.Fatal(err)
	}
	initRepo(t, seed)
	if output, err := exec.Command("git", "-C", seed, "branch", "-M", "main").CombinedOutput(); err != nil {
		t.Fatalf("git branch -M failed: %v (%s)", err, string(output))
	}
	cmd := exec.Command("git", "remote", "add", "origin", remote)
	cmd.Dir = seed
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git remote add failed: %v (%s)", err, string(output))
	}
	if output, err := exec.Command("git", "-C", seed, "config", "core.hooksPath", "/dev/null").CombinedOutput(); err != nil {
		t.Fatalf("git config core.hooksPath failed: %v (%s)", err, string(output))
	}
	commitFile(t, seed, "README.md", "seed\n", "seed")
	cmd = exec.Command("git", "push", "-u", "origin", "main")
	cmd.Dir = seed
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git push seed failed: %v (%s)", err, string(output))
	}

	stale := filepath.Join(tmp, "stale")
	if output, err := exec.Command("git", "clone", "-b", "main", remote, stale).CombinedOutput(); err != nil {
		t.Fatalf("git clone failed: %v (%s)", err, string(output))
	}

	expectedSHA := commitFile(t, seed, "README.md", "seed\nnext\n", "next")
	cmd = exec.Command("git", "push", "origin", "main")
	cmd.Dir = seed
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git push next failed: %v (%s)", err, string(output))
	}

	status, err := EnsureRepoSync(stale, "origin", true, false)
	if err != nil {
		t.Fatal(err)
	}
	if status.Pushed {
		t.Fatalf("expected fast-forward without an extra push, got %+v", status)
	}
	if !status.Synced || status.LocalSHA != expectedSHA || status.RemoteSHA != expectedSHA {
		t.Fatalf("expected stale branch to fast-forward to %s, got %+v", expectedSHA, status)
	}
	if got := gitOutput(t, stale, "rev-parse", "HEAD"); got != expectedSHA {
		t.Fatalf("expected local head %s, got %s", expectedSHA, got)
	}
	if got := gitOutput(t, stale, "rev-parse", "origin/main"); got != expectedSHA {
		t.Fatalf("expected origin/main %s, got %s", expectedSHA, got)
	}
}

func TestInspectRepoSyncMarksDirtyWorktree(t *testing.T) {
	tmp := t.TempDir()
	remote := filepath.Join(tmp, "remote.git")
	if output, err := exec.Command("git", "init", "--bare", remote).CombinedOutput(); err != nil {
		t.Fatalf("git init --bare failed: %v (%s)", err, string(output))
	}

	repo := filepath.Join(tmp, "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	initRepo(t, repo)
	cmd := exec.Command("git", "remote", "add", "origin", remote)
	cmd.Dir = repo
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git remote add failed: %v (%s)", err, string(output))
	}
	commitFile(t, repo, "README.md", "hello\n", "initial commit")
	if _, err := EnsureRepoSync(repo, "origin", true, false); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	status, err := InspectRepoSync(repo, "origin")
	if err != nil {
		t.Fatal(err)
	}
	if !status.Dirty || !status.Synced {
		t.Fatalf("expected dirty synced status, got %+v", status)
	}
}
