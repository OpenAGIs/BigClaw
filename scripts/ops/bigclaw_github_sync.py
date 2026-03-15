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

from bigclaw.github_sync import GitSyncError, ensure_repo_sync, inspect_repo_sync, install_git_hooks


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Install or verify BigClaw GitHub sync hooks.")
    parser.add_argument("command", choices=["install", "status", "sync"])
    parser.add_argument("--repo", default=".", help="Repository path to inspect.")
    parser.add_argument("--remote", default="origin", help="Git remote to verify.")
    parser.add_argument("--hooks-path", default=".githooks", help="Repository-relative hooks directory.")
    parser.add_argument("--allow-dirty", action="store_true", help="Do not fail sync when the worktree has uncommitted changes.")
    parser.add_argument("--require-clean", action="store_true", help="Return a non-zero exit code when the worktree is dirty.")
    parser.add_argument("--require-synced", action="store_true", help="Return a non-zero exit code when local and remote SHAs differ.")
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
    repo = Path(args.repo).resolve()

    try:
        if args.command == "install":
            hooks_dir = install_git_hooks(repo, hooks_path=args.hooks_path)
            emit({"status": "installed", "repo": str(repo), "hooks_path": str(hooks_dir)}, args.json)
            return 0

        if args.command == "status":
            status = inspect_repo_sync(repo, remote=args.remote)
            payload = {"status": "ok", **status.to_dict()}
            emit(payload, args.json)
            if args.require_clean and status.dirty:
                return 1
            if args.require_synced and not status.synced:
                return 1
            return 0

        status = ensure_repo_sync(repo, remote=args.remote, auto_push=True, allow_dirty=args.allow_dirty)
        payload = {"status": "ok", **status.to_dict()}
        emit(payload, args.json)
        if args.require_clean and status.dirty:
            return 1
        if args.require_synced and not status.synced:
            return 1
        return 0
    except GitSyncError as exc:
        emit({"status": "error", "error": str(exc), "repo": str(repo)}, args.json)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
