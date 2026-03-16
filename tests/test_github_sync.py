import subprocess
from pathlib import Path

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


def test_install_git_hooks_configures_core_hooks_path(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    repo.mkdir()
    init_repo(repo)
    hooks_dir = repo / ".githooks"
    hooks_dir.mkdir()
    hook_path = hooks_dir / "post-commit"
    hook_path.write_text("#!/usr/bin/env bash\nexit 0\n")

    installed = install_git_hooks(repo)

    assert installed == hooks_dir
    assert git(repo, "config", "--get", "core.hooksPath") == ".githooks"
    assert hook_path.stat().st_mode & 0o111


def test_ensure_repo_sync_pushes_head_to_origin(tmp_path: Path) -> None:
    remote = tmp_path / "remote.git"
    subprocess.run(["git", "init", "--bare", str(remote)], check=True, capture_output=True, text=True)

    repo = tmp_path / "repo"
    repo.mkdir()
    init_repo(repo)
    git(repo, "remote", "add", "origin", str(remote))
    local_sha = commit_file(repo, "README.md", "hello\n", "initial commit")

    status = ensure_repo_sync(repo)

    assert status.pushed is True
    assert status.synced is True
    assert status.local_sha == local_sha
    assert status.remote_sha == local_sha


def test_inspect_repo_sync_marks_dirty_worktree(tmp_path: Path) -> None:
    remote = tmp_path / "remote.git"
    subprocess.run(["git", "init", "--bare", str(remote)], check=True, capture_output=True, text=True)

    repo = tmp_path / "repo"
    repo.mkdir()
    init_repo(repo)
    git(repo, "remote", "add", "origin", str(remote))
    commit_file(repo, "README.md", "hello\n", "initial commit")
    ensure_repo_sync(repo)

    (repo / "README.md").write_text("dirty\n")
    status = inspect_repo_sync(repo)

    assert status.dirty is True
    assert status.synced is True


def test_ensure_repo_sync_fast_forwards_clean_branch_before_push(tmp_path: Path) -> None:
    remote = tmp_path / "remote.git"
    subprocess.run(["git", "init", "--bare", str(remote)], check=True, capture_output=True, text=True)

    seed = tmp_path / "seed"
    seed.mkdir()
    init_repo(seed)
    git(seed, "branch", "-M", "main")
    git(seed, "remote", "add", "origin", str(remote))
    git(seed, "config", "core.hooksPath", "/dev/null")
    commit_file(seed, "README.md", "seed\n", "seed")
    git(seed, "push", "-u", "origin", "main")

    stale = tmp_path / "stale"
    subprocess.run(["git", "clone", "-b", "main", str(remote), str(stale)], check=True, capture_output=True, text=True)

    commit_file(seed, "README.md", "seed\nnext\n", "next")
    git(seed, "push", "origin", "main")

    status = ensure_repo_sync(stale)

    assert status.synced is True
    assert status.local_sha == status.remote_sha
    assert status.pushed is False
    assert git(stale, "rev-parse", "HEAD") == git(stale, "rev-parse", "origin/main")


def test_ensure_repo_sync_skips_pushing_clean_branch_at_origin_default_head(tmp_path: Path) -> None:
    remote = tmp_path / "remote.git"
    subprocess.run(
        ["git", "init", "--bare", "--initial-branch=main", str(remote)],
        check=True,
        capture_output=True,
        text=True,
    )

    seed = tmp_path / "seed"
    seed.mkdir()
    init_repo(seed)
    git(seed, "branch", "-M", "main")
    git(seed, "remote", "add", "origin", str(remote))
    commit_file(seed, "README.md", "seed\n", "seed")
    git(seed, "push", "-u", "origin", "main")

    repo = tmp_path / "repo"
    subprocess.run(["git", "clone", "-b", "main", str(remote), str(repo)], check=True, capture_output=True, text=True)
    git(repo, "checkout", "-b", "symphony/OPE-321")

    inspected = inspect_repo_sync(repo)
    status = ensure_repo_sync(repo)

    assert inspected.remote_exists is False
    assert inspected.synced is True
    assert status.remote_exists is False
    assert status.synced is True
    assert status.pushed is False
    assert git(repo, "ls-remote", "--heads", "origin", "symphony/OPE-321") == ""
    assert git(repo, "rev-parse", "HEAD") == git(repo, "rev-parse", "origin/main")
