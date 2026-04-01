import argparse
import json
import warnings
from pathlib import Path

from .observability import RepoSyncAudit
from .reports import render_repo_sync_audit_report, write_report
from .service import run_server


_LEGACY_RUNTIME_GUIDANCE = (
    "bigclaw-go is the sole implementation mainline for active development; "
    "the legacy Python runtime surface remains migration-only."
)


def _warn_legacy_runtime_surface(surface: str, replacement: str) -> str:
    message = f"{surface} is frozen for migration-only use. {_LEGACY_RUNTIME_GUIDANCE} Use {replacement} instead."
    warnings.warn(message, DeprecationWarning, stacklevel=2)
    return message


def main() -> None:
    _warn_legacy_runtime_surface("python -m bigclaw", "bash scripts/ops/bigclawctl or go run ./bigclaw-go/cmd/bigclawd")
    parser = argparse.ArgumentParser(prog="bigclaw", description="BigClaw developer utilities")
    sub = parser.add_subparsers(dest="command")

    serve = sub.add_parser("serve", help="Run legacy migration-only static web server")
    serve.add_argument("--host", default="127.0.0.1")
    serve.add_argument("--port", type=int, default=8008)
    serve.add_argument("--dir", default="reports")

    repo_sync_audit = sub.add_parser("repo-sync-audit", help="Render a repo sync and PR freshness audit report")
    repo_sync_audit.add_argument("--input", required=True, help="Path to JSON payload for RepoSyncAudit")
    repo_sync_audit.add_argument("--output", required=True, help="Path to markdown report output")

    args = parser.parse_args()

    if args.command == "serve":
        run_server(host=args.host, port=args.port, directory=args.dir)
        return

    if args.command == "repo-sync-audit":
        payload = json.loads(Path(args.input).read_text())
        audit = RepoSyncAudit.from_dict(payload)
        write_report(args.output, render_repo_sync_audit_report(audit))
        return

    parser.print_help()


if __name__ == "__main__":
    main()
