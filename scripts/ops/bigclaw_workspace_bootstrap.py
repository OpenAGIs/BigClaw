#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

sys.dont_write_bytecode = True

REPO_ROOT = Path(__file__).resolve().parents[2]
SRC_ROOT = REPO_ROOT / "src"
if str(SRC_ROOT) not in sys.path:
    sys.path.insert(0, str(SRC_ROOT))

from bigclaw.workspace_bootstrap import (
    WorkspaceBootstrapError,
    bootstrap_workspace,
    cleanup_workspace,
)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Bootstrap BigClaw workspaces from a shared local mirror.")
    parser.add_argument("command", choices=["bootstrap", "cleanup"])
    parser.add_argument("--workspace", default=".", help="Workspace path managed by Symphony.")
    parser.add_argument("--issue", default="", help="Linear issue identifier used for the bootstrap branch.")
    parser.add_argument(
        "--repo-url",
        default="git@github.com:OpenAGIs/BigClaw.git",
        help="Canonical remote repository URL.",
    )
    parser.add_argument("--default-branch", default="main", help="Default branch used as the bootstrap base.")
    parser.add_argument(
        "--cache-root",
        default="~/.cache/symphony/bigclaw",
        help="Shared cache root for the bare mirror and seed repository.",
    )
    parser.add_argument("--json", action="store_true", help="Emit machine-readable JSON output.")
    return parser.parse_args()


def emit(payload: dict, as_json: bool) -> None:
    if as_json:
        print(json.dumps(payload, ensure_ascii=False, indent=2))
        return
    for key, value in payload.items():
        print(f"{key}={value}")


def main() -> int:
    args = parse_args()
    workspace = Path(args.workspace).expanduser().resolve()

    try:
        if args.command == "bootstrap":
            status = bootstrap_workspace(
                workspace=workspace,
                issue_identifier=args.issue,
                repo_url=args.repo_url,
                default_branch=args.default_branch,
                cache_root=args.cache_root,
            )
        else:
            status = cleanup_workspace(
                workspace=workspace,
                issue_identifier=args.issue,
                repo_url=args.repo_url,
                default_branch=args.default_branch,
                cache_root=args.cache_root,
            )
        emit({"status": "ok", **status.to_dict()}, args.json)
        return 0
    except WorkspaceBootstrapError as exc:
        emit({"status": "error", "workspace": str(workspace), "error": str(exc)}, args.json)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
