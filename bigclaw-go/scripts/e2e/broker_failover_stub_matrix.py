#!/usr/bin/env python3
"""Legacy compatibility shim for bigclawctl automation e2e broker-failover-stub-matrix."""

from __future__ import annotations

import subprocess
import sys
from pathlib import Path


def main() -> int:
    repo_root = Path(__file__).resolve().parents[3]
    command = [
        "go",
        "run",
        "./cmd/bigclawctl",
        "automation",
        "e2e",
        "broker-failover-stub-matrix",
        *sys.argv[1:],
    ]
    return subprocess.call(command, cwd=repo_root / "bigclaw-go")


if __name__ == "__main__":
    raise SystemExit(main())
