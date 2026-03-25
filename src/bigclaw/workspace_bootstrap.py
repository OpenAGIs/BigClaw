from __future__ import annotations

import json
import shutil
import subprocess
from dataclasses import asdict, dataclass
from pathlib import Path
from typing import Any, Sequence
from urllib.parse import urlparse


class WorkspaceBootstrapError(RuntimeError):
    """Raised when the shared-worktree bootstrap flow cannot complete."""


@dataclass
class CacheBootstrapState:
    cache_root: str
    cache_key: str
    mirror_path: str
    seed_path: str
    mirror_created: bool
    seed_created: bool

    def to_dict(self) -> dict[str, Any]:
        return asdict(self)


@dataclass
class SharedBootstrapSnapshot:
    repo_url: str
    repo_locator: str
    default_branch: str
    cache_root: str
    cache_key: str
    mirror_path: str
    seed_path: str
    mirror_created: bool = False
    seed_created: bool = False

    def to_dict(self) -> dict[str, Any]:
        return asdict(self)

    @classmethod
    def from_dict(cls, payload: dict[str, Any]) -> "SharedBootstrapSnapshot":
        return cls(**payload)


@dataclass
class WorkspaceBootstrapStatus:
    workspace: str
    branch: str
    cache_root: str
    cache_key: str
    mirror_path: str
    seed_path: str
    reused: bool
    cache_reused: bool
    clone_suppressed: bool
    mirror_created: bool = False
    seed_created: bool = False
    workspace_mode: str = "worktree_created"
    snapshot_reused: bool = False
    removed: bool = False

    def to_dict(self) -> dict[str, Any]:
        return asdict(self)


@dataclass
class CommandResult:
    stdout: str
    stderr: str
    returncode: int


@dataclass
class WorkspacePrewarmStatus:
    snapshot: SharedBootstrapSnapshot
    workspaces: list[WorkspaceBootstrapStatus]

    def to_dict(self) -> dict[str, Any]:
        return {
            "snapshot": self.snapshot.to_dict(),
            "workspaces": [workspace.to_dict() for workspace in self.workspaces],
            "summary": {
                "workspace_count": len(self.workspaces),
                "snapshot_reused_for_all": all(workspace.snapshot_reused for workspace in self.workspaces),
                "cache_reused_after_first": all(workspace.cache_reused for workspace in self.workspaces[1:]),
                "clone_suppressed_after_first": all(workspace.clone_suppressed for workspace in self.workspaces[1:]),
            },
        }


CACHE_REMOTE = "cache"
BOOTSTRAP_BRANCH_PREFIX = "symphony"
DEFAULT_CACHE_BASE = Path("~/.cache/symphony/repos")


def _run(command: Sequence[str], cwd: Path) -> CommandResult:
    completed = subprocess.run(
        list(command),
        cwd=cwd,
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
        raise WorkspaceBootstrapError(detail)
    return result.stdout


def sanitize_issue_identifier(identifier: str | None) -> str:
    raw = (identifier or "issue").strip() or "issue"
    return "".join(character if character.isalnum() or character in ".-_" else "_" for character in raw)


def bootstrap_branch_name(identifier: str | None) -> str:
    return f"{BOOTSTRAP_BRANCH_PREFIX}/{sanitize_issue_identifier(identifier)}"


def default_cache_base(path: str | Path | None = None) -> Path:
    if path is None:
        return DEFAULT_CACHE_BASE.expanduser().resolve()
    return Path(path).expanduser().resolve()


def normalize_repo_locator(repo_url: str) -> str:
    raw = repo_url.strip()

    if "://" in raw:
        parsed = urlparse(raw)
        locator = f"{parsed.netloc}{parsed.path}"
    elif ":" in raw and "@" in raw.split(":", 1)[0]:
        user_host, repo_path = raw.split(":", 1)
        host = user_host.split("@", 1)[-1]
        locator = f"{host}/{repo_path}"
    else:
        locator = raw

    return locator.strip().rstrip("/").removesuffix(".git")


def repo_cache_key(repo_url: str, cache_key: str | None = None) -> str:
    raw = (cache_key or normalize_repo_locator(repo_url)).strip().lower()
    sanitized = "".join(character if character.isalnum() or character in ".-_" else "-" for character in raw)
    compact = "-".join(segment for segment in sanitized.split("-") if segment)
    return compact or "repo"


def cache_root_for_repo(
    repo_url: str,
    cache_base: str | Path | None = None,
    cache_key: str | None = None,
) -> Path:
    return default_cache_base(cache_base) / repo_cache_key(repo_url, cache_key)


def resolve_cache_root(
    repo_url: str,
    cache_root: str | Path | None = None,
    cache_base: str | Path | None = None,
    cache_key: str | None = None,
) -> Path:
    if cache_root is not None:
        return Path(cache_root).expanduser().resolve()
    return cache_root_for_repo(repo_url, cache_base=cache_base, cache_key=cache_key)


def default_cache_root(path: str | Path | None = None) -> Path:
    return default_cache_base(path)


def _remove_path(path: Path) -> None:
    if path.is_dir() and not path.is_symlink():
        shutil.rmtree(path)
    elif path.exists() or path.is_symlink():
        path.unlink()


def _cache_state(
    repo_url: str,
    repo_cache_root: Path,
    cache_key: str | None = None,
    *,
    mirror_created: bool = False,
    seed_created: bool = False,
) -> CacheBootstrapState:
    return CacheBootstrapState(
        cache_root=str(repo_cache_root),
        cache_key=repo_cache_key(repo_url, cache_key),
        mirror_path=str(repo_cache_root / "mirror.git"),
        seed_path=str(repo_cache_root / "seed"),
        mirror_created=mirror_created,
        seed_created=seed_created,
    )


def _snapshot_from_cache_state(
    repo_url: str,
    default_branch: str,
    cache_state: CacheBootstrapState,
) -> SharedBootstrapSnapshot:
    return SharedBootstrapSnapshot(
        repo_url=repo_url,
        repo_locator=normalize_repo_locator(repo_url),
        default_branch=default_branch,
        cache_root=cache_state.cache_root,
        cache_key=cache_state.cache_key,
        mirror_path=cache_state.mirror_path,
        seed_path=cache_state.seed_path,
        mirror_created=cache_state.mirror_created,
        seed_created=cache_state.seed_created,
    )


def _cache_state_from_snapshot(
    snapshot: SharedBootstrapSnapshot,
    *,
    mirror_created: bool = False,
    seed_created: bool = False,
) -> CacheBootstrapState:
    return CacheBootstrapState(
        cache_root=snapshot.cache_root,
        cache_key=snapshot.cache_key,
        mirror_path=snapshot.mirror_path,
        seed_path=snapshot.seed_path,
        mirror_created=mirror_created,
        seed_created=seed_created,
    )


def _coerce_snapshot(shared_snapshot: SharedBootstrapSnapshot | dict[str, Any]) -> SharedBootstrapSnapshot:
    if isinstance(shared_snapshot, SharedBootstrapSnapshot):
        return shared_snapshot
    return SharedBootstrapSnapshot.from_dict(shared_snapshot)


def _snapshot_ready(snapshot: SharedBootstrapSnapshot) -> bool:
    mirror_path = Path(snapshot.mirror_path)
    seed_path = Path(snapshot.seed_path)
    return (mirror_path / "HEAD").exists() and (seed_path / ".git").exists()


def ensure_mirror(
    repo_url: str,
    cache_root: str | Path | None = None,
    cache_base: str | Path | None = None,
    cache_key: str | None = None,
) -> CacheBootstrapState:
    repo_cache_root = resolve_cache_root(repo_url, cache_root=cache_root, cache_base=cache_base, cache_key=cache_key)
    mirror_path = repo_cache_root / "mirror.git"
    mirror_path.parent.mkdir(parents=True, exist_ok=True)
    mirror_created = False

    if not (mirror_path / "HEAD").exists():
        if mirror_path.exists():
            _remove_path(mirror_path)
        result = subprocess.run(
            ["git", "clone", "--mirror", repo_url, str(mirror_path)],
            text=True,
            capture_output=True,
            check=False,
        )
        if result.returncode != 0:
            detail = result.stderr.strip() or result.stdout.strip() or "git clone --mirror failed"
            raise WorkspaceBootstrapError(detail)
        mirror_created = True
    else:
        _require_git(mirror_path, "remote", "set-url", "origin", repo_url)
        _require_git(mirror_path, "fetch", "--prune", "origin")

    return _cache_state(repo_url, repo_cache_root, cache_key, mirror_created=mirror_created)


def ensure_seed(
    repo_url: str,
    default_branch: str,
    cache_root: str | Path | None = None,
    cache_base: str | Path | None = None,
    cache_key: str | None = None,
) -> CacheBootstrapState:
    cache_state = ensure_mirror(
        repo_url,
        cache_root=cache_root,
        cache_base=cache_base,
        cache_key=cache_key,
    )
    seed_path = Path(cache_state.seed_path)
    seed_created = False

    if not (seed_path / ".git").exists():
        if seed_path.exists():
            _remove_path(seed_path)
        result = subprocess.run(
            ["git", "clone", cache_state.mirror_path, str(seed_path)],
            text=True,
            capture_output=True,
            check=False,
        )
        if result.returncode != 0:
            detail = result.stderr.strip() or result.stdout.strip() or "git clone seed failed"
            raise WorkspaceBootstrapError(detail)
        seed_created = True

    configure_seed_remotes(seed_path, repo_url, Path(cache_state.mirror_path))
    _require_git(seed_path, "fetch", "--prune", CACHE_REMOTE)
    _require_git(seed_path, "worktree", "prune")
    _require_git(seed_path, "checkout", "-B", default_branch, f"{CACHE_REMOTE}/{default_branch}")
    return _cache_state(
        repo_url,
        Path(cache_state.cache_root),
        cache_key,
        mirror_created=cache_state.mirror_created,
        seed_created=seed_created,
    )


def prepare_shared_snapshot(
    repo_url: str,
    default_branch: str = "main",
    cache_root: str | Path | None = None,
    cache_base: str | Path | None = None,
    cache_key: str | None = None,
) -> SharedBootstrapSnapshot:
    cache_state = ensure_seed(
        repo_url,
        default_branch,
        cache_root=cache_root,
        cache_base=cache_base,
        cache_key=cache_key,
    )
    return _snapshot_from_cache_state(repo_url, default_branch, cache_state)


def configure_seed_remotes(seed_path: Path, repo_url: str, mirror_path: Path) -> None:
    remotes = set(_require_git(seed_path, "remote").splitlines())

    if CACHE_REMOTE not in remotes and "origin" in remotes:
        current_origin = _require_git(seed_path, "remote", "get-url", "origin")
        if Path(current_origin).expanduser().resolve() == mirror_path.resolve():
            _require_git(seed_path, "remote", "rename", "origin", CACHE_REMOTE)
            remotes = set(_require_git(seed_path, "remote").splitlines())

    if CACHE_REMOTE not in remotes:
        _require_git(seed_path, "remote", "add", CACHE_REMOTE, str(mirror_path))
    else:
        _require_git(seed_path, "remote", "set-url", CACHE_REMOTE, str(mirror_path))

    remotes = set(_require_git(seed_path, "remote").splitlines())
    if "origin" not in remotes:
        _require_git(seed_path, "remote", "add", "origin", repo_url)
    else:
        _require_git(seed_path, "remote", "set-url", "origin", repo_url)

    _require_git(seed_path, "config", "remote.pushDefault", "origin")


def bootstrap_workspace(
    workspace: str | Path,
    issue_identifier: str | None,
    repo_url: str,
    default_branch: str = "main",
    cache_root: str | Path | None = None,
    cache_base: str | Path | None = None,
    cache_key: str | None = None,
    shared_snapshot: SharedBootstrapSnapshot | dict[str, Any] | None = None,
) -> WorkspaceBootstrapStatus:
    workspace_path = Path(workspace).expanduser().resolve()
    snapshot_reused = shared_snapshot is not None
    if shared_snapshot is None:
        cache_state = ensure_seed(
            repo_url,
            default_branch,
            cache_root=cache_root,
            cache_base=cache_base,
            cache_key=cache_key,
        )
    else:
        snapshot = _coerce_snapshot(shared_snapshot)
        if not _snapshot_ready(snapshot):
            snapshot = prepare_shared_snapshot(
                snapshot.repo_url,
                default_branch=snapshot.default_branch,
                cache_root=snapshot.cache_root,
                cache_key=snapshot.cache_key,
            )
            cache_state = _cache_state_from_snapshot(
                snapshot,
                mirror_created=snapshot.mirror_created,
                seed_created=snapshot.seed_created,
            )
        else:
            cache_state = _cache_state_from_snapshot(snapshot)
        repo_url = snapshot.repo_url
        default_branch = snapshot.default_branch
    seed_path = Path(cache_state.seed_path)
    branch = bootstrap_branch_name(issue_identifier or workspace_path.name)
    cache_reused = not cache_state.mirror_created and not cache_state.seed_created
    clone_suppressed = not cache_state.mirror_created

    git_dir = workspace_path / ".git"
    if git_dir.exists():
        current_branch = _require_git(workspace_path, "branch", "--show-current")
        return WorkspaceBootstrapStatus(
            workspace=str(workspace_path),
            branch=current_branch or branch,
            cache_root=cache_state.cache_root,
            cache_key=cache_state.cache_key,
            mirror_path=cache_state.mirror_path,
            seed_path=cache_state.seed_path,
            reused=True,
            cache_reused=cache_reused,
            clone_suppressed=clone_suppressed,
            mirror_created=cache_state.mirror_created,
            seed_created=cache_state.seed_created,
            workspace_mode="workspace_reused",
            snapshot_reused=snapshot_reused,
        )

    parent = workspace_path.parent
    parent.mkdir(parents=True, exist_ok=True)

    if workspace_path.exists() and workspace_path.is_dir() and any(workspace_path.iterdir()):
        raise WorkspaceBootstrapError(f"Workspace is not empty: {workspace_path}")

    _require_git(seed_path, "worktree", "add", "--force", "-B", branch, str(workspace_path), f"{CACHE_REMOTE}/{default_branch}")

    return WorkspaceBootstrapStatus(
        workspace=str(workspace_path),
        branch=branch,
        cache_root=cache_state.cache_root,
        cache_key=cache_state.cache_key,
        mirror_path=cache_state.mirror_path,
        seed_path=cache_state.seed_path,
        reused=False,
        cache_reused=cache_reused,
        clone_suppressed=clone_suppressed,
        mirror_created=cache_state.mirror_created,
        seed_created=cache_state.seed_created,
        workspace_mode="worktree_created",
        snapshot_reused=snapshot_reused,
    )


def prewarm_workspaces(
    workspaces: Sequence[tuple[str | Path, str | None]],
    repo_url: str,
    default_branch: str = "main",
    cache_root: str | Path | None = None,
    cache_base: str | Path | None = None,
    cache_key: str | None = None,
    shared_snapshot: SharedBootstrapSnapshot | dict[str, Any] | None = None,
) -> WorkspacePrewarmStatus:
    snapshot = (
        _coerce_snapshot(shared_snapshot)
        if shared_snapshot is not None
        else prepare_shared_snapshot(
            repo_url,
            default_branch=default_branch,
            cache_root=cache_root,
            cache_base=cache_base,
            cache_key=cache_key,
        )
    )
    if not _snapshot_ready(snapshot):
        snapshot = prepare_shared_snapshot(
            snapshot.repo_url,
            default_branch=snapshot.default_branch,
            cache_root=snapshot.cache_root,
            cache_key=snapshot.cache_key,
        )

    results = [
        bootstrap_workspace(
            workspace=workspace,
            issue_identifier=issue_identifier,
            repo_url=snapshot.repo_url,
            default_branch=snapshot.default_branch,
            shared_snapshot=snapshot,
        )
        for workspace, issue_identifier in workspaces
    ]
    return WorkspacePrewarmStatus(snapshot=snapshot, workspaces=results)


def cleanup_workspace(
    workspace: str | Path,
    issue_identifier: str | None,
    repo_url: str,
    default_branch: str = "main",
    cache_root: str | Path | None = None,
    cache_base: str | Path | None = None,
    cache_key: str | None = None,
) -> WorkspaceBootstrapStatus:
    workspace_path = Path(workspace).expanduser().resolve()
    repo_cache_root = resolve_cache_root(repo_url, cache_root=cache_root, cache_base=cache_base, cache_key=cache_key)
    cache_state = _cache_state(repo_url, repo_cache_root, cache_key)
    seed_path = Path(cache_state.seed_path)
    mirror_path = Path(cache_state.mirror_path)
    branch = bootstrap_branch_name(issue_identifier or workspace_path.name)

    if not (seed_path / ".git").exists() or not workspace_path.exists():
        return WorkspaceBootstrapStatus(
            workspace=str(workspace_path),
            branch=branch,
            cache_root=cache_state.cache_root,
            cache_key=cache_state.cache_key,
            mirror_path=cache_state.mirror_path,
            seed_path=cache_state.seed_path,
            reused=False,
            cache_reused=seed_path.exists() or mirror_path.exists(),
            clone_suppressed=True,
            workspace_mode="cleanup",
            removed=False,
        )

    configure_seed_remotes(seed_path, repo_url, mirror_path)

    if _git(workspace_path, "rev-parse", "--git-dir").returncode == 0:
        current_branch = _require_git(workspace_path, "branch", "--show-current")
        branch = current_branch or branch

    worktree_list = _require_git(seed_path, "worktree", "list", "--porcelain")
    registered = f"worktree {workspace_path}" in worktree_list
    if registered:
        _require_git(seed_path, "worktree", "remove", "--force", str(workspace_path))
        _require_git(seed_path, "worktree", "prune")

    local_branches = set(_require_git(seed_path, "branch", "--format", "%(refname:short)").splitlines())
    if branch.startswith(f"{BOOTSTRAP_BRANCH_PREFIX}/") and branch in local_branches:
        _require_git(seed_path, "branch", "-D", branch)

    _require_git(seed_path, "checkout", "-B", default_branch, f"{CACHE_REMOTE}/{default_branch}")
    return WorkspaceBootstrapStatus(
        workspace=str(workspace_path),
        branch=branch,
        cache_root=cache_state.cache_root,
        cache_key=cache_state.cache_key,
        mirror_path=cache_state.mirror_path,
        seed_path=cache_state.seed_path,
        reused=False,
        cache_reused=True,
        clone_suppressed=True,
        workspace_mode="cleanup",
        removed=registered,
    )


def status_as_json(status: WorkspaceBootstrapStatus) -> str:
    return json.dumps(status.to_dict(), ensure_ascii=False, indent=2)
