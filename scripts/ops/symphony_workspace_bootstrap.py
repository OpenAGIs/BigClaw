#!/usr/bin/env python3
"""Legacy compatibility shim for the Go workspace command."""

from __future__ import annotations

import subprocess
import sys
from pathlib import Path

repo_root = Path(__file__).resolve().parents[2]


def main() -> int:
    command = ["bash", str(repo_root / "scripts/ops/bigclawctl"), "workspace", *sys.argv[1:]]
    return subprocess.call(command, cwd=repo_root)


if __name__ == "__main__":
    raise SystemExit(main())
