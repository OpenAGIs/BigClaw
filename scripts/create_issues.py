#!/usr/bin/env python3
"""Legacy compatibility shim for the Go create-issues command."""

from __future__ import annotations

import subprocess
import sys
from pathlib import Path


def main() -> int:
    repo_root = Path(__file__).resolve().parents[1]
    command = ["bash", str(repo_root / "scripts/ops/bigclawctl"), "create-issues", *sys.argv[1:]]
    return subprocess.call(command, cwd=repo_root)


if __name__ == "__main__":
    raise SystemExit(main())
