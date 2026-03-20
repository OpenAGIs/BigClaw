import argparse
import json
import sys
from pathlib import Path

from .observability import RepoSyncAudit
from .reports import render_repo_sync_audit_report, write_report
from .service import run_server

LEGACY_ENTRYPOINT_NOTICE = (
    "Legacy Python entrypoint: use scripts/ops/bigclawctl or go run ./cmd/bigclawd for the Go mainline. "
    "Keep python -m bigclaw only for migration or legacy-path validation."
)


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="bigclaw",
        description="BigClaw legacy Python developer utilities",
        epilog=LEGACY_ENTRYPOINT_NOTICE,
    )
    sub = parser.add_subparsers(dest="command")

    serve = sub.add_parser("serve", help="Run local BigClaw static web server")
    serve.add_argument("--host", default="127.0.0.1")
    serve.add_argument("--port", type=int, default=8008)
    serve.add_argument("--dir", default="reports")

    repo_sync_audit = sub.add_parser("repo-sync-audit", help="Render a repo sync and PR freshness audit report")
    repo_sync_audit.add_argument("--input", required=True, help="Path to JSON payload for RepoSyncAudit")
    repo_sync_audit.add_argument("--output", required=True, help="Path to markdown report output")
    return parser


def print_legacy_notice() -> None:
    print(LEGACY_ENTRYPOINT_NOTICE, file=sys.stderr)


def main() -> None:
    parser = build_parser()
    args = parser.parse_args()

    if args.command == "serve":
        print_legacy_notice()
        run_server(host=args.host, port=args.port, directory=args.dir)
        return

    if args.command == "repo-sync-audit":
        print_legacy_notice()
        payload = json.loads(Path(args.input).read_text())
        audit = RepoSyncAudit.from_dict(payload)
        write_report(args.output, render_repo_sync_audit_report(audit))
        return

    parser.print_help()


if __name__ == "__main__":
    main()
