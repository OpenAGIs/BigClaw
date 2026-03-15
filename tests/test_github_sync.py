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
