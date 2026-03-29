from __future__ import annotations

import json
import argparse
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
    removed: bool = False

    def to_dict(self) -> dict[str, Any]:
        return asdict(self)


@dataclass
class CommandResult:
    stdout: str
    stderr: str
    returncode: int


CACHE_REMOTE = "cache"
BOOTSTRAP_BRANCH_PREFIX = "symphony"
DEFAULT_CACHE_BASE = Path("~/.cache/symphony/repos")
DEFAULT_CLI_CACHE_BASE = "~/.cache/symphony/repos"


def build_parser(
    description: str,
    default_repo_url: str,
    default_branch: str,
    default_cache_root: str | None,
    default_cache_base: str,
    default_cache_key: str | None,
) -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=description)
    parser.add_argument("command", choices=["bootstrap", "cleanup"])
    parser.add_argument("--workspace", default=".", help="Workspace path managed by Symphony.")
    parser.add_argument("--issue", default="", help="Linear issue identifier used for the bootstrap branch.")
    parser.add_argument("--repo-url", default=default_repo_url, help="Canonical remote repository URL.")
    parser.add_argument("--default-branch", default=default_branch, help="Default branch used as the bootstrap base.")
    parser.add_argument(
        "--cache-root",
        default=default_cache_root,
        help="Full cache root that contains mirror.git and seed. Overrides --cache-base/--cache-key.",
    )
    parser.add_argument(
        "--cache-base",
        default=default_cache_base,
        help="Base directory that stores per-repo cache roots.",
    )
    parser.add_argument(
        "--cache-key",
        default=default_cache_key,
        help="Optional stable key for the per-repo cache directory.",
    )
    parser.add_argument("--json", action="store_true", help="Emit machine-readable JSON output.")
    return parser


def emit(payload: dict, as_json: bool) -> None:
    if as_json:
        print(json.dumps(payload, ensure_ascii=False, indent=2))
        return
    for key, value in payload.items():
        print(f"{key}={value}")


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


def build_validation_report(
    *,
    repo_url: str,
    workspace_root: str | Path,
    issue_identifiers: Sequence[str],
    default_branch: str = "main",
    cache_root: str | Path | None = None,
    cache_base: str | Path | None = None,
    cache_key: str | None = None,
    cleanup: bool = True,
) -> dict[str, Any]:
    workspace_root_path = Path(workspace_root).expanduser().resolve()
    workspace_root_path.mkdir(parents=True, exist_ok=True)

    bootstrap_results = []
    for issue_identifier in issue_identifiers:
        workspace_path = workspace_root_path / issue_identifier
        status = bootstrap_workspace(
            workspace=workspace_path,
            issue_identifier=issue_identifier,
            repo_url=repo_url,
            default_branch=default_branch,
            cache_root=cache_root,
            cache_base=cache_base,
            cache_key=cache_key,
        )
        bootstrap_results.append(status.to_dict())

    cache_roots = sorted({result["cache_root"] for result in bootstrap_results})
    mirror_paths = sorted({result["mirror_path"] for result in bootstrap_results})
    seed_paths = sorted({result["seed_path"] for result in bootstrap_results})
    cleanup_results = []

    if cleanup:
        for issue_identifier in issue_identifiers:
            workspace_path = workspace_root_path / issue_identifier
            status = cleanup_workspace(
                workspace=workspace_path,
                issue_identifier=issue_identifier,
                repo_url=repo_url,
                default_branch=default_branch,
                cache_root=cache_root,
                cache_base=cache_base,
                cache_key=cache_key,
            )
            cleanup_results.append(status.to_dict())

    return {
        "repo_url": repo_url,
        "default_branch": default_branch,
        "workspace_root": str(workspace_root_path),
        "issue_identifiers": list(issue_identifiers),
        "bootstrap_results": bootstrap_results,
        "cleanup_results": cleanup_results,
        "summary": {
            "workspace_count": len(bootstrap_results),
            "unique_cache_roots": cache_roots,
            "unique_mirror_paths": mirror_paths,
            "unique_seed_paths": seed_paths,
            "single_cache_root_reused": len(cache_roots) == 1,
            "single_mirror_reused": len(mirror_paths) == 1,
            "single_seed_reused": len(seed_paths) == 1,
            "mirror_creations": sum(1 for result in bootstrap_results if result["mirror_created"]),
            "seed_creations": sum(1 for result in bootstrap_results if result["seed_created"]),
            "clone_suppressed_after_first": all(result["clone_suppressed"] for result in bootstrap_results[1:]),
            "cache_reused_after_first": all(result["cache_reused"] for result in bootstrap_results[1:]),
            "all_workspaces_created_via_worktree": all(
                result["workspace_mode"] in {"worktree_created", "workspace_reused"}
                for result in bootstrap_results
            ),
            "cleanup_preserved_cache": bool(bootstrap_results)
            and Path(bootstrap_results[0]["mirror_path"]).exists()
            and Path(bootstrap_results[0]["seed_path"]).joinpath(".git").exists(),
        },
    }


def render_validation_markdown(report: dict[str, Any]) -> str:
    summary = report["summary"]
    lines = [
        "# Symphony bootstrap cache validation",
        "",
        f"- Repo: `{report['repo_url']}`",
        f"- Workspace root: `{report['workspace_root']}`",
        f"- Workspaces: `{summary['workspace_count']}`",
        f"- Single cache root reused: `{summary['single_cache_root_reused']}`",
        f"- Mirror creations: `{summary['mirror_creations']}`",
        f"- Seed creations: `{summary['seed_creations']}`",
        f"- Clone suppressed after first workspace: `{summary['clone_suppressed_after_first']}`",
        f"- Cleanup preserved cache: `{summary['cleanup_preserved_cache']}`",
        "",
        "## Bootstrap Results",
        "",
    ]

    for result in report["bootstrap_results"]:
        lines.extend(
            [
                f"- `{result['workspace']}`",
                f"  - `cache_root={result['cache_root']}`",
                f"  - `cache_key={result['cache_key']}`",
                f"  - `workspace_mode={result['workspace_mode']}`",
                f"  - `cache_reused={result['cache_reused']}`",
                f"  - `clone_suppressed={result['clone_suppressed']}`",
                f"  - `mirror_created={result['mirror_created']}`",
                f"  - `seed_created={result['seed_created']}`",
            ]
        )

    return "\n".join(lines) + "\n"


def write_validation_report(report: dict[str, Any], path: str | Path) -> Path:
    target = Path(path).expanduser().resolve()
    target.parent.mkdir(parents=True, exist_ok=True)

    if target.suffix.lower() == ".md":
        target.write_text(render_validation_markdown(report))
    else:
        target.write_text(json.dumps(report, ensure_ascii=False, indent=2))
    return target


def main(
    argv: Sequence[str] | None = None,
    *,
    description: str = "Bootstrap Symphony workspaces from a shared local mirror.",
    default_repo_url: str = "",
    default_branch: str = "main",
    default_cache_root: str | None = None,
    default_cache_base: str = DEFAULT_CLI_CACHE_BASE,
    default_cache_key: str | None = None,
) -> int:
    parser = build_parser(
        description=description,
        default_repo_url=default_repo_url,
        default_branch=default_branch,
        default_cache_root=default_cache_root,
        default_cache_base=default_cache_base,
        default_cache_key=default_cache_key,
    )
    args = parser.parse_args(argv)
    workspace = Path(args.workspace).expanduser().resolve()

    try:
        payload = dict(
            workspace=workspace,
            issue_identifier=args.issue,
            repo_url=args.repo_url,
            default_branch=args.default_branch,
            cache_root=args.cache_root,
            cache_base=args.cache_base,
            cache_key=args.cache_key,
        )
        if args.command == "bootstrap":
            status = bootstrap_workspace(**payload)
        else:
            status = cleanup_workspace(**payload)
        emit({"status": "ok", **status.to_dict()}, args.json)
        return 0
    except WorkspaceBootstrapError as exc:
        emit({"status": "error", "workspace": str(workspace), "error": str(exc)}, args.json)
        return 1


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
) -> WorkspaceBootstrapStatus:
    workspace_path = Path(workspace).expanduser().resolve()
    cache_state = ensure_seed(
        repo_url,
        default_branch,
        cache_root=cache_root,
        cache_base=cache_base,
        cache_key=cache_key,
    )
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
    )


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
