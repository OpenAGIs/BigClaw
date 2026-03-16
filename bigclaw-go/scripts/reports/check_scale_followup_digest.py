#!/usr/bin/env python3
from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path


REQUIRED_REPORTS = {
    "docs/reports/scale-validation-follow-up-digest.md": [
        "`10k` queue reliability matrix",
        "production-grade capacity certification",
        "local `2000x24` soak run",
    ],
    "docs/reports/queue-reliability-report.md": [
        "docs/reports/scale-validation-follow-up-digest.md",
    ],
    "docs/reports/benchmark-readiness-report.md": [
        "docs/reports/scale-validation-follow-up-digest.md",
    ],
    "docs/reports/long-duration-soak-report.md": [
        "docs/reports/scale-validation-follow-up-digest.md",
    ],
    "docs/reports/review-readiness.md": [
        "docs/reports/scale-validation-follow-up-digest.md",
    ],
    "docs/reports/issue-coverage.md": [
        "docs/reports/scale-validation-follow-up-digest.md",
    ],
}


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Validate the scale follow-up digest links and caveat wording.")
    parser.add_argument(
        "--go-root",
        default=Path(__file__).resolve().parents[2],
        type=Path,
        help="Path to the bigclaw-go repository root.",
    )
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    go_root = args.go_root.resolve()
    failures: list[str] = []

    for relative_path, required_snippets in REQUIRED_REPORTS.items():
        report_path = go_root / relative_path
        if not report_path.exists():
            failures.append(f"missing report: {relative_path}")
            continue
        content = report_path.read_text(encoding="utf-8")
        for snippet in required_snippets:
            if snippet not in content:
                failures.append(f"missing snippet in {relative_path}: {snippet}")

    digest_path = go_root / "docs/reports/scale-validation-follow-up-digest.md"
    if digest_path.exists():
        digest = digest_path.read_text(encoding="utf-8")
        referenced_paths = sorted(set(re.findall(r"`(docs/reports/[^`]+)`", digest)))
        for reference in referenced_paths:
            if not (go_root / reference).exists():
                failures.append(f"digest references missing path: {reference}")

    if failures:
        for failure in failures:
            print(f"FAIL: {failure}", file=sys.stderr)
        return 1

    print("scale follow-up digest check passed")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
