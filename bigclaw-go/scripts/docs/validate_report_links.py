#!/usr/bin/env python3
"""Validate that repo-relative file references in selected report docs exist."""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

REFERENCE_RE = re.compile(
    r"`((?:(?:(?:\.\./)+)?(?:cmd|docs|examples|internal|scripts)/[^`\n*]+?\.(?:go|json|md|py|sh|txt|yaml|yml)))`"
)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Validate repo-relative file references in markdown reports."
    )
    parser.add_argument(
        "documents",
        nargs="*",
        default=[
            "docs/reports/issue-coverage.md",
            "docs/reports/linear-project-sync-summary.md",
            "docs/reports/epic-closure-readiness-report.md",
        ],
        help="Markdown documents relative to the bigclaw-go repo root.",
    )
    return parser.parse_args()


def collect_references(document: Path) -> list[str]:
    text = document.read_text(encoding="utf-8")
    return sorted(set(REFERENCE_RE.findall(text)))


def resolve_reference(repo_root: Path, document: Path, reference: str) -> Path:
    if reference.startswith("../"):
        return (document.parent / reference).resolve()
    return repo_root / reference


def main() -> int:
    args = parse_args()
    repo_root = Path(__file__).resolve().parents[2]
    failures: list[str] = []

    for relative_document in args.documents:
        document = repo_root / relative_document
        if not document.is_file():
            failures.append(f"{relative_document}: document not found")
            continue

        references = collect_references(document)
        missing = [
            reference
            for reference in references
            if not resolve_reference(repo_root, document, reference).exists()
        ]
        if missing:
            failures.extend(f"{relative_document}: missing {reference}" for reference in missing)
            continue

        print(f"{relative_document}: {len(references)} references OK")

    if failures:
        for failure in failures:
            print(failure, file=sys.stderr)
        return 1

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
