from __future__ import annotations

import re
from collections import Counter
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
TARGET_REPORTS = [
    ROOT / "bigclaw-go" / "docs" / "reports" / "issue-coverage.md",
    ROOT / "bigclaw-go" / "docs" / "reports" / "queue-reliability-report.md",
    ROOT / "bigclaw-go" / "docs" / "reports" / "event-bus-reliability-report.md",
    ROOT / "bigclaw-go" / "docs" / "reports" / "multi-subscriber-takeover-validation-report.md",
    ROOT / "bigclaw-go" / "docs" / "reports" / "review-readiness.md",
]
REPORT_PATH_PATTERN = re.compile(r"`(docs/[^`*]+\.(?:md|json))`")


def _resolve_repo_path(report_path: str) -> Path:
    for candidate in (ROOT / report_path, ROOT / "bigclaw-go" / report_path):
        if candidate.exists():
            return candidate
    raise AssertionError(f"missing referenced report path: {report_path}")


def test_target_reports_reference_existing_repo_paths() -> None:
    for report in TARGET_REPORTS:
        content = report.read_text()
        references = REPORT_PATH_PATTERN.findall(content)
        assert references, f"{report} should reference at least one repo-native report path"
        for reference in references:
            resolved = _resolve_repo_path(reference)
            assert resolved.exists(), f"{report} references missing path {reference}"


def test_target_reports_do_not_repeat_bullets_verbatim() -> None:
    for report in TARGET_REPORTS:
        bullets = [
            line.strip()
            for line in report.read_text().splitlines()
            if line.lstrip().startswith("- ")
        ]
        duplicates = sorted(line for line, count in Counter(bullets).items() if count > 1)
        assert not duplicates, f"{report} repeats bullet lines: {duplicates}"
