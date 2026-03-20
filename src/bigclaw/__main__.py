import argparse
import json
from pathlib import Path

from .deprecation import warn_legacy_runtime_surface
from .observability import RepoSyncAudit
from .reports import render_repo_sync_audit_report, write_report
from .service import run_server


def main() -> None:
    warn_legacy_runtime_surface("python -m bigclaw", "bash scripts/ops/bigclawctl or go run ./bigclaw-go/cmd/bigclawd")
    parser = argparse.ArgumentParser(prog="bigclaw", description="BigClaw developer utilities")
    sub = parser.add_subparsers(dest="command")

    serve = sub.add_parser("serve", help="Run local BigClaw static web server")
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
