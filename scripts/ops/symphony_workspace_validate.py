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

from bigclaw.workspace_bootstrap_validation import build_validation_report, write_validation_report


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Validate shared-cache Symphony workspace bootstrap behavior.")
    parser.add_argument("--repo-url", required=True, help="Canonical Git remote to validate.")
    parser.add_argument("--workspace-root", required=True, help="Root directory for temporary issue workspaces.")
    parser.add_argument("--issues", nargs="+", required=True, help="Issue identifiers to bootstrap sequentially.")
    parser.add_argument("--default-branch", default="main", help="Default branch used as the bootstrap base.")
    parser.add_argument("--cache-root", default=None, help="Optional full cache root override.")
    parser.add_argument("--cache-base", default="~/.cache/symphony/repos", help="Base directory for per-repo caches.")
    parser.add_argument("--cache-key", default=None, help="Optional stable cache key override.")
    parser.add_argument("--report-file", default="", help="Optional path to write JSON or Markdown validation output.")
    parser.add_argument("--no-cleanup", action="store_true", help="Preserve workspaces after validation.")
    parser.add_argument("--json", action="store_true", help="Emit machine-readable JSON output.")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    report = build_validation_report(
        repo_url=args.repo_url,
        workspace_root=args.workspace_root,
        issue_identifiers=args.issues,
        default_branch=args.default_branch,
        cache_root=args.cache_root,
        cache_base=args.cache_base,
        cache_key=args.cache_key,
        cleanup=not args.no_cleanup,
    )

    if args.report_file:
        report_path = write_validation_report(report, args.report_file)
        report["report_file"] = str(report_path)

    if args.json:
        print(json.dumps(report, ensure_ascii=False, indent=2))
    else:
        print(f"workspace_count={report['summary']['workspace_count']}")
        print(f"single_cache_root_reused={report['summary']['single_cache_root_reused']}")
        print(f"mirror_creations={report['summary']['mirror_creations']}")
        print(f"seed_creations={report['summary']['seed_creations']}")
        print(f"clone_suppressed_after_first={report['summary']['clone_suppressed_after_first']}")
        if args.report_file:
            print(f"report_file={report['report_file']}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
