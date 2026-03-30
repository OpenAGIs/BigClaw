#!/usr/bin/env python3
"""Legacy compatibility shim for the Go workspace validate command."""

from __future__ import annotations

import subprocess
import sys
from pathlib import Path

repo_root = Path(__file__).resolve().parents[2]


def translate_workspace_validate_args(forwarded: list[str]) -> list[str]:
    translated: list[str] = []
    i = 0
    while i < len(forwarded):
        arg = forwarded[i]
        if arg == "--report-file":
            translated.extend(["--report", forwarded[i + 1]])
            i += 2
            continue
        if arg.startswith("--report-file="):
            translated.append("--report=" + arg.split("=", 1)[1])
            i += 1
            continue
        if arg == "--no-cleanup":
            translated.append("--cleanup=false")
            i += 1
            continue
        if arg == "--issues":
            issues: list[str] = []
            i += 1
            while i < len(forwarded) and not forwarded[i].startswith("-"):
                issues.append(forwarded[i])
                i += 1
            translated.extend(["--issues", ",".join(issues)])
            continue
        translated.append(arg)
        i += 1
    return translated


def main() -> int:
    repo_root = Path(__file__).resolve().parents[2]
    translated = translate_workspace_validate_args(sys.argv[1:])
    command = ["bash", str(repo_root / "scripts/ops/bigclawctl"), "workspace", "validate", *translated]
    return subprocess.call(command, cwd=repo_root)


if __name__ == "__main__":
    raise SystemExit(main())
