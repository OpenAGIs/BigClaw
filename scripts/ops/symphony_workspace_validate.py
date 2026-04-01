#!/usr/bin/env python3
"""Legacy compatibility shim for the Go workspace validate command."""

from __future__ import annotations

import subprocess
import sys
from pathlib import Path

def translate_workspace_validate_args(forwarded: list[str]) -> list[str]:
    translated: list[str] = []
    index = 0
    while index < len(forwarded):
        arg = forwarded[index]
        if arg == "--report-file":
            translated.extend(["--report", forwarded[index + 1]])
            index += 2
            continue
        if arg.startswith("--report-file="):
            translated.append("--report=" + arg.split("=", 1)[1])
            index += 1
            continue
        if arg == "--no-cleanup":
            translated.append("--cleanup=false")
            index += 1
            continue
        if arg == "--issues":
            issues: list[str] = []
            index += 1
            while index < len(forwarded) and not forwarded[index].startswith("-"):
                issues.append(forwarded[index])
                index += 1
            translated.extend(["--issues", ",".join(issues)])
            continue
        translated.append(arg)
        index += 1
    return translated


def main() -> int:
    repo_root = Path(__file__).resolve().parents[2]
    translated = translate_workspace_validate_args(sys.argv[1:])
    command = ["bash", str(repo_root / "scripts/ops/bigclawctl"), "workspace", "validate", *translated]
    return subprocess.call(command, cwd=repo_root)


if __name__ == "__main__":
    raise SystemExit(main())
