#!/usr/bin/env python3
import pathlib
import sys


REQUIRED = {
    "docs/reports/multi-node-coordination-report.md": {
        "snippets": [
            "No dedicated leader-election layer exists yet",
            "docs/reports/event-bus-reliability-report.md",
            "docs/reports/multi-subscriber-takeover-validation-report.md",
            "docs/reports/review-readiness.md",
            "docs/reports/issue-coverage.md",
        ],
        "artifacts": [
            "docs/reports/multi-node-shared-queue-report.json",
            "docs/reports/event-bus-reliability-report.md",
            "docs/reports/multi-subscriber-takeover-validation-report.md",
            "docs/reports/multi-subscriber-takeover-validation-report.json",
            "docs/reports/review-readiness.md",
            "docs/reports/issue-coverage.md",
        ],
    },
    "docs/reports/review-readiness.md": {
        "snippets": [
            "docs/reports/multi-node-coordination-report.md",
            "docs/reports/event-bus-reliability-report.md",
            "docs/reports/multi-subscriber-takeover-validation-report.md",
        ],
        "artifacts": [
            "docs/reports/multi-node-coordination-report.md",
            "docs/reports/event-bus-reliability-report.md",
            "docs/reports/multi-subscriber-takeover-validation-report.md",
        ],
    },
    "docs/reports/issue-coverage.md": {
        "snippets": [
            "docs/reports/multi-node-coordination-report.md",
            "docs/reports/event-bus-reliability-report.md",
            "docs/reports/multi-subscriber-takeover-validation-report.md",
            "docs/reports/multi-subscriber-takeover-validation-report.json",
        ],
        "artifacts": [
            "docs/reports/multi-node-coordination-report.md",
            "docs/reports/event-bus-reliability-report.md",
            "docs/reports/multi-subscriber-takeover-validation-report.md",
            "docs/reports/multi-subscriber-takeover-validation-report.json",
        ],
    },
    "docs/reports/event-bus-reliability-report.md": {
        "snippets": [
            "docs/reports/multi-node-coordination-report.md",
            "docs/reports/multi-subscriber-takeover-validation-report.md",
            "docs/reports/multi-subscriber-takeover-validation-report.json",
        ],
        "artifacts": [
            "docs/reports/multi-node-coordination-report.md",
            "docs/reports/multi-subscriber-takeover-validation-report.md",
            "docs/reports/multi-subscriber-takeover-validation-report.json",
        ],
    },
    "docs/reports/multi-subscriber-takeover-validation-report.md": {
        "snippets": [
            "docs/reports/multi-node-coordination-report.md",
            "docs/reports/event-bus-reliability-report.md",
            "docs/reports/review-readiness.md",
            "docs/reports/issue-coverage.md",
        ],
        "artifacts": [
            "docs/reports/multi-node-shared-queue-report.json",
            "docs/reports/multi-node-coordination-report.md",
            "docs/reports/event-bus-reliability-report.md",
            "docs/reports/review-readiness.md",
            "docs/reports/issue-coverage.md",
            "docs/reports/multi-subscriber-takeover-validation-report.json",
        ],
    },
}


def main() -> int:
    repo_root = pathlib.Path(__file__).resolve().parents[2]
    errors = []

    for relative_path, spec in REQUIRED.items():
        path = repo_root / relative_path
        if not path.is_file():
            errors.append(f"missing file: {relative_path}")
            continue
        text = path.read_text(encoding="utf-8")
        for snippet in spec["snippets"]:
            if snippet not in text:
                errors.append(f"{relative_path}: missing snippet {snippet!r}")
        for artifact in spec["artifacts"]:
            if not (repo_root / artifact).exists():
                errors.append(f"{relative_path}: missing artifact target {artifact}")

    if errors:
        for error in errors:
            print(error, file=sys.stderr)
        return 1

    print("coordination digest links OK")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
