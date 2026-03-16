from __future__ import annotations

import argparse
import json
from pathlib import Path
from typing import Sequence

from bigclaw.workspace_bootstrap import (
    WorkspaceBootstrapError,
    bootstrap_workspace,
    cleanup_workspace,
)


DEFAULT_CACHE_BASE = "~/.cache/symphony/repos"


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


def main(
    argv: Sequence[str] | None = None,
    *,
    description: str = "Bootstrap Symphony workspaces from a shared local mirror.",
    default_repo_url: str = "",
    default_branch: str = "main",
    default_cache_root: str | None = None,
    default_cache_base: str = DEFAULT_CACHE_BASE,
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
