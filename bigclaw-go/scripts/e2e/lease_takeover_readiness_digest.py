#!/usr/bin/env python3
import argparse
import json
import pathlib
import sys
from typing import Iterable


REPO_ROOT = pathlib.Path(__file__).resolve().parents[2]
LEASE_REPORT = REPO_ROOT / "docs/reports/lease-recovery-report.md"
COORDINATION_REPORT = REPO_ROOT / "docs/reports/multi-node-coordination-report.md"
COORDINATION_JSON = REPO_ROOT / "docs/reports/multi-node-shared-queue-report.json"
TAKEOVER_REPORT = REPO_ROOT / "docs/reports/multi-subscriber-takeover-validation-report.md"
TAKEOVER_JSON = REPO_ROOT / "docs/reports/multi-subscriber-takeover-validation-report.json"
OUTPUT_PATH = REPO_ROOT / "docs/reports/lease-takeover-readiness-digest.md"


def read_text(path: pathlib.Path) -> str:
    return path.read_text(encoding="utf-8")


def read_json(path: pathlib.Path) -> dict:
    return json.loads(read_text(path))


def section_lines(markdown: str, heading: str) -> list[str]:
    lines = markdown.splitlines()
    capture = False
    collected: list[str] = []
    target = f"## {heading}"
    for line in lines:
        if line.startswith("## "):
            if capture:
                break
            capture = line.strip() == target
            continue
        if capture:
            collected.append(line)
    return collected


def section_bullets(markdown: str, heading: str) -> list[str]:
    bullets: list[str] = []
    for line in section_lines(markdown, heading):
        stripped = line.strip()
        if stripped.startswith("- "):
            bullets.append(stripped[2:])
    return bullets


def first_value(markdown: str, heading: str, label: str) -> str:
    prefix = f"- {label}: "
    for line in section_lines(markdown, heading):
        stripped = line.strip()
        if stripped.startswith(prefix):
            return stripped[len(prefix) :].strip()
    raise ValueError(f"missing {label!r} under {heading!r}")


def render_bullets(items: Iterable[str]) -> list[str]:
    return [f"- {item}" for item in items]


def build_digest() -> str:
    lease_markdown = read_text(LEASE_REPORT)
    coordination_markdown = read_text(COORDINATION_REPORT)
    coordination = read_json(COORDINATION_JSON)
    takeover_markdown = read_text(TAKEOVER_REPORT)
    takeover = read_json(TAKEOVER_JSON)

    lease_behaviors = section_bullets(lease_markdown, "Verified Behaviors")
    lease_result = section_bullets(lease_markdown, "Current Result")
    takeover_result = section_bullets(takeover_markdown, "Current Result")
    run_date = first_value(coordination_markdown, "Scope", "Run date")
    command = first_value(coordination_markdown, "Scope", "Command")
    required_sections = [f"`{section}`" for section in takeover["required_report_sections"]]

    duplicate_started = len(coordination.get("duplicate_started_tasks", []))
    duplicate_completed = len(coordination.get("duplicate_completed_tasks", []))
    missing_completed = len(coordination.get("missing_completed_tasks", []))

    lines: list[str] = [
        "# Lease And Takeover Readiness Digest",
        "",
        "## Scope",
        "",
        "This digest consolidates the current lease recovery, shared-queue coordination, and takeover-readiness evidence for `OPE-246` into one reviewer-facing report.",
        "",
        "## Current Evidence Snapshot",
        "",
        f"- Lease recovery report: `docs/reports/lease-recovery-report.md`",
        f"- Multi-node coordination report: `docs/reports/multi-node-coordination-report.md`",
        f"- Coordination artifact: `docs/reports/multi-node-shared-queue-report.json`",
        f"- Takeover validation contract: `docs/reports/multi-subscriber-takeover-validation-report.md`",
        f"- Takeover validation matrix: `docs/reports/multi-subscriber-takeover-validation-report.json`",
        "",
        "## Lease Recovery Evidence",
        "",
        *render_bullets(lease_behaviors),
        "",
        "## Shared-Queue Coordination Evidence",
        "",
        f"- Run date: `{run_date}`",
        f"- Command: `{command}`",
        f"- Total tasks: `{coordination['count']}`",
        f"- Submitted by `node-a`: `{coordination['submitted_by_node']['node-a']}`",
        f"- Submitted by `node-b`: `{coordination['submitted_by_node']['node-b']}`",
        f"- Completed by `node-a`: `{coordination['completed_by_node']['node-a']}`",
        f"- Completed by `node-b`: `{coordination['completed_by_node']['node-b']}`",
        f"- Cross-node completions: `{coordination['cross_node_completions']}`",
        f"- Duplicate `task.started`: `{duplicate_started}`",
        f"- Duplicate `task.completed`: `{duplicate_completed}`",
        f"- Missing terminal completions: `{missing_completed}`",
        f"- Overall result: `{'pass' if coordination.get('all_ok') else 'fail'}`",
        "",
        "## Takeover Readiness Contract",
        "",
        f"- Current status: `{takeover['status']}`",
        f"- Source ticket: `{takeover['ticket']}`",
        f"- Generated matrix timestamp: `{takeover['generated_at']}`",
        f"- Required report sections: {', '.join(required_sections)}",
        "",
        "The takeover matrix is planning-ready evidence. It defines the assertions and report shape reviewers should expect once shared multi-node subscriber-group takeover automation exists, but it does not claim the end-to-end fault injection is implemented today.",
        "",
        "## Scenario Assertions",
        "",
    ]

    for scenario in takeover["scenarios"]:
        lines.extend(
            [
                f"### `{scenario['id']}`",
                "",
                f"- Title: {scenario['title']}",
                f"- Target gap: {scenario['fault']['target_gap']}",
                "- Audit assertions:",
                *[f"  - {item}" for item in scenario["audit_assertions"]],
                "- Checkpoint assertions:",
                *[f"  - {item}" for item in scenario["checkpoint_assertions"]],
                "- Replay assertions:",
                *[f"  - {item}" for item in scenario["replay_assertions"]],
                "- Current blockers:",
                *[f"  - {item}" for item in scenario["implementation_blockers"]],
                "",
            ]
        )

    lines.extend(
        [
            "## Review Guidance",
            "",
            *render_bullets(lease_result),
            *render_bullets(takeover_result),
            "- Current shared-queue evidence proves lease expiry recovery and two-node coordination, but not durable cross-node subscriber-group takeover fencing.",
            "- `docs/reports/event-bus-reliability-report.md` documents subscriber-group lease concepts and current API/reporting boundaries; reviewers should treat the takeover matrix here as the stricter cross-node readiness contract.",
            "",
            "## Validation",
            "",
            "- Regenerate with `python3 scripts/e2e/lease_takeover_readiness_digest.py --write`.",
            "- Verify consistency with `python3 scripts/e2e/lease_takeover_readiness_digest.py --check`.",
            "",
        ]
    )
    return "\n".join(lines)


def check_digest(expected: str) -> int:
    current = read_text(OUTPUT_PATH).rstrip("\n")
    expected = expected.rstrip("\n")
    if current == expected:
        print(f"ok: {OUTPUT_PATH.relative_to(REPO_ROOT)}")
        return 0
    print(
        f"mismatch: regenerate {OUTPUT_PATH.relative_to(REPO_ROOT)} with "
        "python3 scripts/e2e/lease_takeover_readiness_digest.py --write",
        file=sys.stderr,
    )
    return 1


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Generate or validate the lease/takeover readiness digest for OPE-246."
    )
    parser.add_argument(
        "--write",
        action="store_true",
        help="Write the generated digest to docs/reports/lease-takeover-readiness-digest.md",
    )
    parser.add_argument(
        "--check",
        action="store_true",
        help="Fail if the on-disk digest does not match the generated output",
    )
    parser.add_argument(
        "--print",
        dest="print_output",
        action="store_true",
        help="Print the generated digest to stdout",
    )
    args = parser.parse_args()

    if not args.write and not args.check and not args.print_output:
        parser.error("choose at least one of --write, --check, or --print")

    digest = build_digest()
    if args.write:
        OUTPUT_PATH.write_text(digest + "\n", encoding="utf-8")
    if args.print_output:
        print(digest)
    if args.check:
        return check_digest(digest)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
