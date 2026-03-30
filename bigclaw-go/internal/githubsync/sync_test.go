package githubsync

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestInstallGitHooksSkipsRewriteWhenHooksPathAlreadyMatches(t *testing.T) {
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

	if _, err := InstallGitHooks(repo, ".githooks"); err != nil {
		t.Fatal(err)
	}
	lockPath := filepath.Join(repo, ".git", "config.lock")
	if err := os.WriteFile(lockPath, []byte("held"), 0o644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(lockPath)

	if _, err := InstallGitHooks(repo, ".githooks"); err != nil {
		t.Fatalf("expected existing hooks path to avoid config rewrite, got %v", err)
	}
}

func TestInstallGitHooksRetriesTransientConfigLock(t *testing.T) {
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

	lockPath := filepath.Join(repo, ".git", "config.lock")
	if err := os.WriteFile(lockPath, []byte("held"), 0o644); err != nil {
		t.Fatal(err)
	}
	go func() {
		time.Sleep(40 * time.Millisecond)
		_ = os.Remove(lockPath)
	}()

	if _, err := InstallGitHooks(repo, ".githooks"); err != nil {
		t.Fatalf("expected transient config lock to be retried, got %v", err)
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
	if !status.Dirty || !status.Synced || !status.Pushed {
		t.Fatalf("expected dirty synced status, got %+v", status)
	}
}

func TestInspectRepoSyncDetachedHeadReportsDefaultBranchSync(t *testing.T) {
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
	detach := exec.Command("git", "checkout", "--detach", "HEAD")
	detach.Dir = repo
	if output, err := detach.CombinedOutput(); err != nil {
		t.Fatalf("git checkout --detach failed: %v (%s)", err, string(output))
	}

	status, err := InspectRepoSync(repo, "origin")
	if err != nil {
		t.Fatal(err)
	}
	if !status.Detached || status.Branch != "HEAD" {
		t.Fatalf("expected detached HEAD branch status, got %+v", status)
	}
	if status.RemoteExists || status.Pushed {
		t.Fatalf("expected detached status to omit remote branch info, got %+v", status)
	}
	if !status.Synced {
		t.Fatalf("expected detached checkout to be synced to remote default branch, got %+v", status)
	}
}

func TestEnsureRepoSyncRefusesAutoPushWhenDetachedAndUnsynced(t *testing.T) {
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
	first := commitFile(t, repo, "README.md", "hello\n", "initial commit")
	if _, err := EnsureRepoSync(repo, "origin", true, false); err != nil {
		t.Fatal(err)
	}
	remoteDefault := gitOutput(t, repo, "remote", "show", "origin")
	defaultBranch := ""
	for _, line := range strings.Split(remoteDefault, "\n") {
		if strings.HasPrefix(line, "  HEAD branch: ") {
			defaultBranch = strings.TrimSpace(strings.TrimPrefix(line, "  HEAD branch: "))
		}
	}
	if defaultBranch == "" {
		t.Fatalf("expected remote show origin to include default branch, got:\n%s", remoteDefault)
	}

	// Advance local history without pushing, then detach at that new commit.
	second := commitFile(t, repo, "tracked.txt", "v2\n", "second commit")
	if second == first {
		t.Fatalf("expected second commit to differ from first")
	}
	detach := exec.Command("git", "checkout", "--detach", "HEAD")
	detach.Dir = repo
	if output, err := detach.CombinedOutput(); err != nil {
		t.Fatalf("git checkout --detach failed: %v (%s)", err, string(output))
	}

	if _, err := EnsureRepoSync(repo, "origin", true, false); err == nil {
		t.Fatalf("expected ensure repo sync to refuse auto-push on detached unsynced HEAD")
	}

	// Remote default branch should remain at the first commit.
	if got := gitOutput(t, repo, "ls-remote", remote, defaultBranch); !strings.HasPrefix(got, first) {
		t.Fatalf("expected remote default branch %s to remain at %s, got %s", defaultBranch, first, got)
	}
}

func TestEnsureRepoSyncPushesDirtyWorktreeWhenRemoteIsBehind(t *testing.T) {
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
	head := commitFile(t, repo, "tracked.txt", "version-2\n", "second commit")
	if err := os.WriteFile(filepath.Join(repo, "local-issues.json"), []byte("{\"issues\":[]}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	status, err := EnsureRepoSync(repo, "origin", true, true)
	if err != nil {
		t.Fatalf("ensure dirty sync: %v", err)
	}
	if !status.Dirty || !status.Pushed || !status.Synced {
		t.Fatalf("expected dirty pushed synced status, got %+v", status)
	}
	if status.LocalSHA != head || status.RemoteSHA != head {
		t.Fatalf("expected remote to match dirty HEAD %s, got %+v", head, status)
	}
}

func TestEnsureRepoSyncRejectsDirtyWorktreeWhenRemoteMoved(t *testing.T) {
	tmp := t.TempDir()
	remote := filepath.Join(tmp, "remote.git")
	if output, err := exec.Command("git", "init", "--bare", remote).CombinedOutput(); err != nil {
		t.Fatalf("git init --bare failed: %v (%s)", err, string(output))
	}

	repoA := filepath.Join(tmp, "repo-a")
	if err := os.MkdirAll(repoA, 0o755); err != nil {
		t.Fatal(err)
	}
	initRepo(t, repoA)
	cmd := exec.Command("git", "remote", "add", "origin", remote)
	cmd.Dir = repoA
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git remote add failed: %v (%s)", err, string(output))
	}
	commitFile(t, repoA, "README.md", "hello\n", "initial commit")
	if _, err := EnsureRepoSync(repoA, "origin", true, false); err != nil {
		t.Fatal(err)
	}

	repoB := filepath.Join(tmp, "repo-b")
	if output, err := exec.Command("git", "clone", remote, repoB).CombinedOutput(); err != nil {
		t.Fatalf("git clone failed: %v (%s)", err, string(output))
	}
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = repoB
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git config user.email failed: %v (%s)", err, string(output))
	}
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = repoB
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git config user.name failed: %v (%s)", err, string(output))
	}
	commitFile(t, repoB, "REMOTE.md", "remote\n", "remote advance")
	push := exec.Command("git", "push", "origin", "HEAD")
	push.Dir = repoB
	if output, err := push.CombinedOutput(); err != nil {
		t.Fatalf("git push failed: %v (%s)", err, string(output))
	}

	if err := os.WriteFile(filepath.Join(repoA, "local-issues.json"), []byte("{\"issues\":[]}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := EnsureRepoSync(repoA, "origin", true, true)
	if err == nil || !strings.Contains(err.Error(), "remote branch moved while working tree is dirty") {
		t.Fatalf("expected dirty remote moved error, got %v", err)
	}
}

func configureCloneIdentity(t *testing.T, repo string) {
	t.Helper()
	for _, args := range [][]string{
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

func TestInspectRepoSyncReportsAheadWhenLocalHasUnpushedCommits(t *testing.T) {
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

	commitFile(t, repo, "tracked.txt", "v2\n", "second commit")
	status, err := InspectRepoSync(repo, "origin")
	if err != nil {
		t.Fatal(err)
	}
	if status.Synced {
		t.Fatalf("expected status to be unsynced after local-only commit, got %+v", status)
	}
	if !status.RelationKnown || status.Ahead != 1 || status.Behind != 0 || status.Diverged {
		t.Fatalf("expected ahead-only relation, got %+v", status)
	}
}

func TestInspectRepoSyncReportsBehindWhenRemoteAdvanced(t *testing.T) {
	tmp := t.TempDir()
	remote := filepath.Join(tmp, "remote.git")
	if output, err := exec.Command("git", "init", "--bare", remote).CombinedOutput(); err != nil {
		t.Fatalf("git init --bare failed: %v (%s)", err, string(output))
	}

	repoA := filepath.Join(tmp, "repo-a")
	if err := os.MkdirAll(repoA, 0o755); err != nil {
		t.Fatal(err)
	}
	initRepo(t, repoA)
	cmd := exec.Command("git", "remote", "add", "origin", remote)
	cmd.Dir = repoA
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git remote add failed: %v (%s)", err, string(output))
	}
	commitFile(t, repoA, "README.md", "hello\n", "initial commit")
	if _, err := EnsureRepoSync(repoA, "origin", true, false); err != nil {
		t.Fatal(err)
	}

	repoB := filepath.Join(tmp, "repo-b")
	if output, err := exec.Command("git", "clone", remote, repoB).CombinedOutput(); err != nil {
		t.Fatalf("git clone failed: %v (%s)", err, string(output))
	}
	configureCloneIdentity(t, repoB)
	commitFile(t, repoB, "REMOTE.md", "remote\n", "remote advance")
	push := exec.Command("git", "push", "origin", "HEAD")
	push.Dir = repoB
	if output, err := push.CombinedOutput(); err != nil {
		t.Fatalf("git push failed: %v (%s)", err, string(output))
	}

	status, err := InspectRepoSync(repoA, "origin")
	if err != nil {
		t.Fatal(err)
	}
	if status.Synced {
		t.Fatalf("expected status to be unsynced after remote-only commit, got %+v", status)
	}
	if !status.RelationKnown || status.Ahead != 0 || status.Behind != 1 || status.Diverged {
		t.Fatalf("expected behind-only relation, got %+v", status)
	}
}

func TestInspectRepoSyncReportsDivergedWhenLocalAndRemoteBothMoved(t *testing.T) {
	tmp := t.TempDir()
	remote := filepath.Join(tmp, "remote.git")
	if output, err := exec.Command("git", "init", "--bare", remote).CombinedOutput(); err != nil {
		t.Fatalf("git init --bare failed: %v (%s)", err, string(output))
	}

	repoA := filepath.Join(tmp, "repo-a")
	if err := os.MkdirAll(repoA, 0o755); err != nil {
		t.Fatal(err)
	}
	initRepo(t, repoA)
	cmd := exec.Command("git", "remote", "add", "origin", remote)
	cmd.Dir = repoA
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git remote add failed: %v (%s)", err, string(output))
	}
	commitFile(t, repoA, "README.md", "hello\n", "initial commit")
	if _, err := EnsureRepoSync(repoA, "origin", true, false); err != nil {
		t.Fatal(err)
	}

	// Local-only commit (ahead).
	commitFile(t, repoA, "LOCAL.md", "local\n", "local advance")

	// Remote-only commit from a separate clone (behind).
	repoB := filepath.Join(tmp, "repo-b")
	if output, err := exec.Command("git", "clone", remote, repoB).CombinedOutput(); err != nil {
		t.Fatalf("git clone failed: %v (%s)", err, string(output))
	}
	configureCloneIdentity(t, repoB)
	commitFile(t, repoB, "REMOTE.md", "remote\n", "remote advance")
	push := exec.Command("git", "push", "origin", "HEAD")
	push.Dir = repoB
	if output, err := push.CombinedOutput(); err != nil {
		t.Fatalf("git push failed: %v (%s)", err, string(output))
	}

	status, err := InspectRepoSync(repoA, "origin")
	if err != nil {
		t.Fatal(err)
	}
	if status.Synced {
		t.Fatalf("expected status to be unsynced after diverging commits, got %+v", status)
	}
	if !status.RelationKnown || status.Ahead != 1 || status.Behind != 1 || !status.Diverged {
		t.Fatalf("expected diverged relation, got %+v", status)
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
	branchRename := exec.Command("git", "branch", "-M", "main")
	branchRename.Dir = seed
	if output, err := branchRename.CombinedOutput(); err != nil {
		t.Fatalf("git branch -M main failed: %v (%s)", err, string(output))
	}
	addOrigin := exec.Command("git", "remote", "add", "origin", remote)
	addOrigin.Dir = seed
	if output, err := addOrigin.CombinedOutput(); err != nil {
		t.Fatalf("git remote add failed: %v (%s)", err, string(output))
	}
	configHooks := exec.Command("git", "config", "core.hooksPath", "/dev/null")
	configHooks.Dir = seed
	if output, err := configHooks.CombinedOutput(); err != nil {
		t.Fatalf("git config core.hooksPath failed: %v (%s)", err, string(output))
	}
	commitFile(t, seed, "README.md", "seed\n", "seed")
	pushSeed := exec.Command("git", "push", "-u", "origin", "main")
	pushSeed.Dir = seed
	if output, err := pushSeed.CombinedOutput(); err != nil {
		t.Fatalf("git push -u origin main failed: %v (%s)", err, string(output))
	}

	stale := filepath.Join(tmp, "stale")
	if output, err := exec.Command("git", "clone", "-b", "main", remote, stale).CombinedOutput(); err != nil {
		t.Fatalf("git clone failed: %v (%s)", err, string(output))
	}

	commitFile(t, seed, "README.md", "seed\nnext\n", "next")
	pushNext := exec.Command("git", "push", "origin", "main")
	pushNext.Dir = seed
	if output, err := pushNext.CombinedOutput(); err != nil {
		t.Fatalf("git push origin main failed: %v (%s)", err, string(output))
	}

	status, err := EnsureRepoSync(stale, "origin", true, false)
	if err != nil {
		t.Fatal(err)
	}
	if !status.Synced || !status.RelationKnown || status.Ahead != 0 || status.Behind != 0 || status.Diverged {
		t.Fatalf("expected fast-forwarded clean branch to converge with remote, got %+v", status)
	}
	if status.LocalSHA != status.RemoteSHA {
		t.Fatalf("expected local and remote SHAs to match after fast-forward, got %+v", status)
	}
	if gotHead, gotRemote := gitOutput(t, stale, "rev-parse", "HEAD"), gitOutput(t, stale, "rev-parse", "origin/main"); gotHead != gotRemote {
		t.Fatalf("expected local HEAD and origin/main to match, got HEAD=%s origin/main=%s", gotHead, gotRemote)
	}
}

func TestEnsureRepoSyncSkipsPushingCleanBranchAtOriginDefaultHead(t *testing.T) {
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
	branchRename := exec.Command("git", "branch", "-M", "main")
	branchRename.Dir = seed
	if output, err := branchRename.CombinedOutput(); err != nil {
		t.Fatalf("git branch -M main failed: %v (%s)", err, string(output))
	}
	addOrigin := exec.Command("git", "remote", "add", "origin", remote)
	addOrigin.Dir = seed
	if output, err := addOrigin.CombinedOutput(); err != nil {
		t.Fatalf("git remote add failed: %v (%s)", err, string(output))
	}
	commitFile(t, seed, "README.md", "seed\n", "seed")
	pushSeed := exec.Command("git", "push", "-u", "origin", "main")
	pushSeed.Dir = seed
	if output, err := pushSeed.CombinedOutput(); err != nil {
		t.Fatalf("git push -u origin main failed: %v (%s)", err, string(output))
	}

	repo := filepath.Join(tmp, "repo")
	if output, err := exec.Command("git", "clone", "-b", "main", remote, repo).CombinedOutput(); err != nil {
		t.Fatalf("git clone failed: %v (%s)", err, string(output))
	}
	checkoutBranch := exec.Command("git", "checkout", "-b", "symphony/OPE-321")
	checkoutBranch.Dir = repo
	if output, err := checkoutBranch.CombinedOutput(); err != nil {
		t.Fatalf("git checkout -b symphony/OPE-321 failed: %v (%s)", err, string(output))
	}

	inspected, err := InspectRepoSync(repo, "origin")
	if err != nil {
		t.Fatal(err)
	}
	status, err := EnsureRepoSync(repo, "origin", true, false)
	if err != nil {
		t.Fatal(err)
	}

	if inspected.RemoteExists || !inspected.Synced {
		t.Fatalf("expected clean issue branch at remote default head to be treated as synced without remote branch, got %+v", inspected)
	}
	if status.RemoteExists || !status.Synced || status.Pushed {
		t.Fatalf("expected ensure sync to skip push for default-head fallback, got %+v", status)
	}
	if got := gitOutput(t, repo, "ls-remote", "--heads", "origin", "symphony/OPE-321"); got != "" {
		t.Fatalf("expected no remote issue branch, got %q", got)
	}
	if gotHead, gotRemote := gitOutput(t, repo, "rev-parse", "HEAD"), gitOutput(t, repo, "rev-parse", "origin/main"); gotHead != gotRemote {
		t.Fatalf("expected local HEAD to stay at origin/main, got HEAD=%s origin/main=%s", gotHead, gotRemote)
	}
}
