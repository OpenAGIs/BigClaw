from __future__ import annotations

import stat
import subprocess
from dataclasses import asdict, dataclass, field
from pathlib import Path
from typing import Any, Dict, List, Protocol


@dataclass
class RepoCommit:
    commit_hash: str
    title: str
    author: str = ""
    parent_hashes: List[str] = field(default_factory=list)
    metadata: Dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> Dict[str, Any]:
        return {
            "commit_hash": self.commit_hash,
            "title": self.title,
            "author": self.author,
            "parent_hashes": list(self.parent_hashes),
            "metadata": dict(self.metadata),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "RepoCommit":
        return cls(
            commit_hash=str(data["commit_hash"]),
            title=str(data.get("title", "")),
            author=str(data.get("author", "")),
            parent_hashes=[str(item) for item in data.get("parent_hashes", [])],
            metadata=dict(data.get("metadata", {})),
        )


@dataclass
class CommitLineage:
    root_hash: str
    lineage: List[RepoCommit] = field(default_factory=list)
    children: Dict[str, List[str]] = field(default_factory=dict)
    leaves: List[str] = field(default_factory=list)

    def to_dict(self) -> Dict[str, Any]:
        return {
            "root_hash": self.root_hash,
            "lineage": [item.to_dict() for item in self.lineage],
            "children": {key: list(value) for key, value in self.children.items()},
            "leaves": list(self.leaves),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "CommitLineage":
        return cls(
            root_hash=str(data.get("root_hash", "")),
            lineage=[RepoCommit.from_dict(item) for item in data.get("lineage", [])],
            children={str(key): [str(v) for v in value] for key, value in dict(data.get("children", {})).items()},
            leaves=[str(item) for item in data.get("leaves", [])],
        )


@dataclass
class CommitDiff:
    left_hash: str
    right_hash: str
    files_changed: int
    insertions: int
    deletions: int
    summary: str = ""

    def to_dict(self) -> Dict[str, Any]:
        return {
            "left_hash": self.left_hash,
            "right_hash": self.right_hash,
            "files_changed": self.files_changed,
            "insertions": self.insertions,
            "deletions": self.deletions,
            "summary": self.summary,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "CommitDiff":
        return cls(
            left_hash=str(data.get("left_hash", "")),
            right_hash=str(data.get("right_hash", "")),
            files_changed=int(data.get("files_changed", 0)),
            insertions=int(data.get("insertions", 0)),
            deletions=int(data.get("deletions", 0)),
            summary=str(data.get("summary", "")),
        )


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


def _run(command: List[str], repo: Path) -> CommandResult:
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


def _remote_default_branch(repo: Path, remote: str) -> str:
    symbolic_ref = _git(repo, "symbolic-ref", "--quiet", f"refs/remotes/{remote}/HEAD")
    if symbolic_ref.returncode == 0 and symbolic_ref.stdout:
        prefix = f"refs/remotes/{remote}/"
        if symbolic_ref.stdout.startswith(prefix):
            return symbolic_ref.stdout[len(prefix) :]

    symref_result = _git(repo, "ls-remote", "--symref", remote, "HEAD")
    if symref_result.returncode != 0:
        detail = symref_result.stderr or symref_result.stdout or f"git ls-remote --symref failed for {remote}/HEAD"
        raise GitSyncError(detail)

    for line in symref_result.stdout.splitlines():
        if line.startswith("ref: ") and line.endswith("\tHEAD"):
            ref = line.split()[1]
            prefix = "refs/heads/"
            if ref.startswith(prefix):
                return ref[len(prefix) :]

    raise GitSyncError(f"Could not determine default branch for remote {remote}")


def _remote_branch_sha(repo: Path, remote: str, branch: str) -> str:
    local_ref = _git(repo, "rev-parse", f"refs/remotes/{remote}/{branch}")
    if local_ref.returncode == 0 and local_ref.stdout:
        return local_ref.stdout

    remote_result = _git(repo, "ls-remote", "--heads", remote, branch)
    if remote_result.returncode != 0:
        detail = remote_result.stderr or remote_result.stdout or f"git ls-remote failed for {remote}/{branch}"
        raise GitSyncError(detail)

    return remote_result.stdout.split()[0] if remote_result.stdout else ""


def _matches_remote_default_branch(repo: Path, remote: str, local_sha: str) -> bool:
    try:
        default_branch = _remote_default_branch(repo, remote)
        default_sha = _remote_branch_sha(repo, remote, default_branch)
    except GitSyncError:
        return False

    return bool(default_sha) and local_sha == default_sha


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

    if not remote_exists and _matches_remote_default_branch(repo_path, remote, local_sha):
        synced = True

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

    if status.remote_exists and not status.synced:
        fetch_result = _git(repo_path, "fetch", remote, status.branch)
        if fetch_result.returncode != 0:
            detail = fetch_result.stderr or fetch_result.stdout or f"git fetch {remote} {status.branch} failed"
            raise GitSyncError(detail)

        ff_result = _git(repo_path, "pull", "--ff-only", remote, status.branch)
        if ff_result.returncode != 0:
            detail = ff_result.stderr or ff_result.stdout or f"git pull --ff-only {remote} {status.branch} failed"
            raise GitSyncError(detail)
        status = inspect_repo_sync(repo_path, remote=remote)

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


class RepoGatewayClient(Protocol):
    def push_bundle(self, repo_space_id: str, bundle_ref: str) -> Dict[str, Any]: ...

    def fetch_bundle(self, repo_space_id: str, bundle_ref: str) -> Dict[str, Any]: ...

    def list_commits(self, repo_space_id: str) -> List[Dict[str, Any]]: ...

    def get_commit(self, repo_space_id: str, commit_hash: str) -> Dict[str, Any]: ...

    def get_children(self, repo_space_id: str, commit_hash: str) -> List[str]: ...

    def get_lineage(self, repo_space_id: str, commit_hash: str) -> Dict[str, Any]: ...

    def get_leaves(self, repo_space_id: str, commit_hash: str) -> List[str]: ...

    def diff(self, repo_space_id: str, left_hash: str, right_hash: str) -> Dict[str, Any]: ...


@dataclass(frozen=True)
class RepoGatewayError:
    code: str
    message: str
    retryable: bool = False

    def to_dict(self) -> Dict[str, Any]:
        return {"code": self.code, "message": self.message, "retryable": self.retryable}


def normalize_gateway_error(error: Exception) -> RepoGatewayError:
    message = str(error).lower()
    if "timeout" in message:
        return RepoGatewayError(code="timeout", message=str(error), retryable=True)
    if "not found" in message:
        return RepoGatewayError(code="not_found", message=str(error), retryable=False)
    return RepoGatewayError(code="gateway_error", message=str(error), retryable=False)


def normalize_commit(payload: Dict[str, Any]) -> RepoCommit:
    return RepoCommit.from_dict(payload)


def normalize_lineage(payload: Dict[str, Any]) -> CommitLineage:
    return CommitLineage.from_dict(payload)


def normalize_diff(payload: Dict[str, Any]) -> CommitDiff:
    return CommitDiff.from_dict(payload)


def repo_audit_payload(*, actor: str, action: str, outcome: str, commit_hash: str, repo_space_id: str) -> Dict[str, Any]:
    return {
        "actor": actor,
        "action": action,
        "outcome": outcome,
        "commit_hash": commit_hash,
        "repo_space_id": repo_space_id,
    }
