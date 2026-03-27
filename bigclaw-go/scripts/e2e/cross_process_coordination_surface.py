#!/usr/bin/env python3
import argparse
import json
import pathlib
import subprocess
import sys
from typing import Any


def _paths() -> tuple[pathlib.Path, pathlib.Path]:
    go_root = pathlib.Path(__file__).resolve().parents[2]
    repo_root = go_root.parent
    return go_root, repo_root


def build_report() -> dict[str, Any]:
    go_root, repo_root = _paths()
    result = subprocess.run(
        [
            "go",
            "run",
            "./scripts/e2e/cross_process_coordination_surface.go",
            "--repo-root",
            str(repo_root),
            "--pretty",
        ],
        cwd=go_root,
        check=True,
        capture_output=True,
        text=True,
    )
    return json.loads(result.stdout)


def main() -> None:
    parser = argparse.ArgumentParser(description="Generate the cross-process coordination capability surface report")
    parser.add_argument("--output", default="bigclaw-go/docs/reports/cross-process-coordination-capability-surface.json")
    parser.add_argument("--pretty", action="store_true")
    args = parser.parse_args()

    go_root, repo_root = _paths()
    cmd = [
        "go",
        "run",
        "./scripts/e2e/cross_process_coordination_surface.go",
        "--repo-root",
        str(repo_root),
        "--output",
        args.output,
    ]
    if args.pretty:
        cmd.append("--pretty")
    raise SystemExit(subprocess.run(cmd, cwd=go_root).returncode)


if __name__ == "__main__":
    main()
