package regression

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestLane8PythonGitHubSyncContractStaysAligned(t *testing.T) {
	goRepoRoot := repoRoot(t)
	repoRoot := filepath.Clean(filepath.Join(goRepoRoot, ".."))
	scriptPath := filepath.Join(t.TempDir(), "github_sync_contract.py")
	script := `import json
import subprocess
import sys
import tempfile
from pathlib import Path

repo_root = Path(sys.argv[1])
sys.path.insert(0, str(repo_root / "src"))

from bigclaw.github_sync import ensure_repo_sync, inspect_repo_sync, install_git_hooks


def git(repo: Path, *args: str) -> str:
    completed = subprocess.run(
        ["git", *args],
        cwd=repo,
        text=True,
        capture_output=True,
        check=True,
    )
    return completed.stdout.strip()


def init_repo(repo: Path) -> None:
    git(repo, "init")
    git(repo, "config", "user.email", "test@example.com")
    git(repo, "config", "user.name", "Test User")


def commit_file(repo: Path, name: str, content: str, message: str) -> str:
    (repo / name).write_text(content)
    git(repo, "add", name)
    git(repo, "commit", "-m", message)
    return git(repo, "rev-parse", "HEAD")


with tempfile.TemporaryDirectory() as td:
    td = Path(td)
    repo = td / "repo"
    repo.mkdir()
    init_repo(repo)
    hooks_dir = repo / ".githooks"
    hooks_dir.mkdir()
    hook_path = hooks_dir / "post-commit"
    hook_path.write_text("#!/usr/bin/env bash\nexit 0\n")
    installed = install_git_hooks(repo)
    hooks = {
        "installed_name": Path(installed).name,
        "config": git(repo, "config", "--get", "core.hooksPath") == ".githooks",
        "executable": bool(hook_path.stat().st_mode & 0o111),
    }

with tempfile.TemporaryDirectory() as td:
    td = Path(td)
    remote = td / "remote.git"
    subprocess.run(["git", "init", "--bare", str(remote)], check=True, capture_output=True, text=True)
    repo = td / "repo"
    repo.mkdir()
    init_repo(repo)
    git(repo, "remote", "add", "origin", str(remote))
    local_sha = commit_file(repo, "README.md", "hello\n", "initial commit")
    status = ensure_repo_sync(repo)
    push = {
        "pushed": status.pushed,
        "synced": status.synced,
        "local_sha": status.local_sha,
        "remote_sha": status.remote_sha,
        "expected_sha": local_sha,
    }

with tempfile.TemporaryDirectory() as td:
    td = Path(td)
    remote = td / "remote.git"
    subprocess.run(["git", "init", "--bare", str(remote)], check=True, capture_output=True, text=True)
    repo = td / "repo"
    repo.mkdir()
    init_repo(repo)
    git(repo, "remote", "add", "origin", str(remote))
    commit_file(repo, "README.md", "hello\n", "initial commit")
    ensure_repo_sync(repo)
    (repo / "README.md").write_text("dirty\n")
    status = inspect_repo_sync(repo)
    dirty = {"dirty": status.dirty, "synced": status.synced}

with tempfile.TemporaryDirectory() as td:
    td = Path(td)
    remote = td / "remote.git"
    subprocess.run(["git", "init", "--bare", str(remote)], check=True, capture_output=True, text=True)
    seed = td / "seed"
    seed.mkdir()
    init_repo(seed)
    git(seed, "branch", "-M", "main")
    git(seed, "remote", "add", "origin", str(remote))
    git(seed, "config", "core.hooksPath", "/dev/null")
    commit_file(seed, "README.md", "seed\n", "seed")
    git(seed, "push", "-u", "origin", "main")
    stale = td / "stale"
    subprocess.run(["git", "clone", "-b", "main", str(remote), str(stale)], check=True, capture_output=True, text=True)
    commit_file(seed, "README.md", "seed\nnext\n", "next")
    git(seed, "push", "origin", "main")
    status = ensure_repo_sync(stale)
    fast_forward = {
        "synced": status.synced,
        "sha_equal": status.local_sha == status.remote_sha,
        "pushed": status.pushed,
        "head_matches_remote": git(stale, "rev-parse", "HEAD") == git(stale, "rev-parse", "origin/main"),
    }

with tempfile.TemporaryDirectory() as td:
    td = Path(td)
    remote = td / "remote.git"
    subprocess.run(
        ["git", "init", "--bare", "--initial-branch=main", str(remote)],
        check=True,
        capture_output=True,
        text=True,
    )
    seed = td / "seed"
    seed.mkdir()
    init_repo(seed)
    git(seed, "branch", "-M", "main")
    git(seed, "remote", "add", "origin", str(remote))
    commit_file(seed, "README.md", "seed\n", "seed")
    git(seed, "push", "-u", "origin", "main")
    repo = td / "repo"
    subprocess.run(["git", "clone", "-b", "main", str(remote), str(repo)], check=True, capture_output=True, text=True)
    git(repo, "checkout", "-b", "symphony/OPE-321")
    inspected = inspect_repo_sync(repo)
    status = ensure_repo_sync(repo)
    default_head = {
        "inspected_remote_exists": inspected.remote_exists,
        "inspected_synced": inspected.synced,
        "remote_exists": status.remote_exists,
        "synced": status.synced,
        "pushed": status.pushed,
        "ls_remote_empty": git(repo, "ls-remote", "--heads", "origin", "symphony/OPE-321") == "",
        "head_matches_origin_main": git(repo, "rev-parse", "HEAD") == git(repo, "rev-parse", "origin/main"),
    }

print(json.dumps({
    "hooks": hooks,
    "push": push,
    "dirty": dirty,
    "fast_forward": fast_forward,
    "default_head": default_head,
}))
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o644); err != nil {
		t.Fatalf("write github sync contract script: %v", err)
	}

	cmd := exec.Command("python3", scriptPath, repoRoot)
	cmd.Dir = goRepoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run github sync contract script: %v\n%s", err, string(output))
	}

	var decoded struct {
		Hooks struct {
			InstalledName string `json:"installed_name"`
			Config        bool   `json:"config"`
			Executable    bool   `json:"executable"`
		} `json:"hooks"`
		Push struct {
			Pushed      bool   `json:"pushed"`
			Synced      bool   `json:"synced"`
			LocalSHA    string `json:"local_sha"`
			RemoteSHA   string `json:"remote_sha"`
			ExpectedSHA string `json:"expected_sha"`
		} `json:"push"`
		Dirty struct {
			Dirty  bool `json:"dirty"`
			Synced bool `json:"synced"`
		} `json:"dirty"`
		FastForward struct {
			Synced           bool `json:"synced"`
			SHAEqual         bool `json:"sha_equal"`
			Pushed           bool `json:"pushed"`
			HeadMatchesRemote bool `json:"head_matches_remote"`
		} `json:"fast_forward"`
		DefaultHead struct {
			InspectedRemoteExists bool `json:"inspected_remote_exists"`
			InspectedSynced       bool `json:"inspected_synced"`
			RemoteExists          bool `json:"remote_exists"`
			Synced                bool `json:"synced"`
			Pushed                bool `json:"pushed"`
			LsRemoteEmpty         bool `json:"ls_remote_empty"`
			HeadMatchesOriginMain bool `json:"head_matches_origin_main"`
		} `json:"default_head"`
	}
	if err := json.Unmarshal(output, &decoded); err != nil {
		t.Fatalf("decode github sync contract output: %v\n%s", err, string(output))
	}

	if decoded.Hooks.InstalledName != ".githooks" || !decoded.Hooks.Config || !decoded.Hooks.Executable {
		t.Fatalf("unexpected hook installation payload: %+v", decoded.Hooks)
	}
	if !decoded.Push.Pushed || !decoded.Push.Synced || decoded.Push.LocalSHA != decoded.Push.ExpectedSHA || decoded.Push.RemoteSHA != decoded.Push.ExpectedSHA {
		t.Fatalf("unexpected push payload: %+v", decoded.Push)
	}
	if !decoded.Dirty.Dirty || !decoded.Dirty.Synced {
		t.Fatalf("unexpected dirty inspect payload: %+v", decoded.Dirty)
	}
	if !decoded.FastForward.Synced || !decoded.FastForward.SHAEqual || decoded.FastForward.Pushed || !decoded.FastForward.HeadMatchesRemote {
		t.Fatalf("unexpected fast-forward payload: %+v", decoded.FastForward)
	}
	if decoded.DefaultHead.InspectedRemoteExists || !decoded.DefaultHead.InspectedSynced || decoded.DefaultHead.RemoteExists || !decoded.DefaultHead.Synced || decoded.DefaultHead.Pushed || !decoded.DefaultHead.LsRemoteEmpty || !decoded.DefaultHead.HeadMatchesOriginMain {
		t.Fatalf("unexpected default-head payload: %+v", decoded.DefaultHead)
	}
}
