from __future__ import annotations

import stat
import subprocess
from dataclasses import asdict, dataclass
from pathlib import Path
from typing import Any, Sequence


class GitSyncError(RuntimeError):
    """Raised when repository sync automation cannot complete safely."""


@dataclass
class RepoSyncStatus:
    branch: str
    local_sha: str
    remote_sha: str
    dirty: bool
    remote_exists: bool
    synced: bool
    pushed: bool = False

    def to_dict(self) -> dict[str, Any]:
        return asdict(self)


@dataclass
class CommandResult:
    stdout: str
    stderr: str
    returncode: int


EXECUTABLE_BITS = stat.S_IXUSR | stat.S_IXGRP | stat.S_IXOTH


def _run(command: Sequence[str], repo: Path) -> CommandResult:
    completed = subprocess.run(
        list(command),
        cwd=repo,
        text=True,
        capture_output=True,
        check=False,
    )
    return CommandResult(
        stdout=completed.stdout.strip(),
        stderr=completed.stderr.strip(),
        returncode=completed.returncode,
    )


def _git(repo: Path, *args: str) -> CommandResult:
    return _run(["git", *args], repo)


def _require_git(repo: Path, *args: str) -> str:
    result = _git(repo, *args)
    if result.returncode != 0:
        detail = result.stderr or result.stdout or f"git {' '.join(args)} failed"
        raise GitSyncError(detail)
    return result.stdout


def _dirty(repo: Path) -> bool:
    return bool(_require_git(repo, "status", "--porcelain"))


def inspect_repo_sync(repo: Path | str, remote: str = "origin") -> RepoSyncStatus:
    repo_path = Path(repo).resolve()
    branch = _require_git(repo_path, "branch", "--show-current")
    if not branch:
        raise GitSyncError("Detached HEAD does not support issue branch sync automation")

    local_sha = _require_git(repo_path, "rev-parse", "HEAD")
    remote_result = _git(repo_path, "ls-remote", "--heads", remote, branch)
    if remote_result.returncode != 0:
        detail = remote_result.stderr or remote_result.stdout or f"git ls-remote failed for {remote}/{branch}"
        raise GitSyncError(detail)

    remote_sha = remote_result.stdout.split()[0] if remote_result.stdout else ""
    dirty = _dirty(repo_path)
    remote_exists = bool(remote_sha)
    synced = remote_exists and local_sha == remote_sha
    return RepoSyncStatus(
        branch=branch,
        local_sha=local_sha,
        remote_sha=remote_sha,
        dirty=dirty,
        remote_exists=remote_exists,
        synced=synced,
    )


def install_git_hooks(repo: Path | str, hooks_path: str = ".githooks") -> Path:
    repo_path = Path(repo).resolve()
    hooks_dir = repo_path / hooks_path
    if not hooks_dir.is_dir():
        raise GitSyncError(f"Missing hooks directory: {hooks_dir}")

    _require_git(repo_path, "config", "core.hooksPath", hooks_path)
    for hook in hooks_dir.iterdir():
        if hook.is_file():
            hook.chmod(hook.stat().st_mode | EXECUTABLE_BITS)
    return hooks_dir


def ensure_repo_sync(
    repo: Path | str,
    remote: str = "origin",
    auto_push: bool = True,
    allow_dirty: bool = False,
) -> RepoSyncStatus:
    repo_path = Path(repo).resolve()
    status = inspect_repo_sync(repo_path, remote=remote)

    if status.dirty:
        if allow_dirty:
            return status
        raise GitSyncError("Working tree is dirty; commit or stash changes before syncing")

    if not auto_push or status.synced:
        return status

    push_args = ["push", remote, "HEAD"]
    if not status.remote_exists:
        push_args = ["push", "-u", remote, "HEAD"]

    push_result = _git(repo_path, *push_args)
    if push_result.returncode != 0:
        detail = push_result.stderr or push_result.stdout or f"git {' '.join(push_args)} failed"
        raise GitSyncError(detail)

    refreshed = inspect_repo_sync(repo_path, remote=remote)
    refreshed.pushed = True
    if not refreshed.synced:
        raise GitSyncError(
            f"Remote SHA mismatch after push: local={refreshed.local_sha} remote={refreshed.remote_sha or 'missing'}"
        )
    return refreshed
