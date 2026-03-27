#!/usr/bin/env python3
"""Legacy compatibility shim for bigclawctl automation e2e validation-bundle-policy-gate."""

from __future__ import annotations

import subprocess
import sys
from pathlib import Path


def main() -> int:
    go_root = Path(__file__).resolve().parents[2]
    command = [
        "go",
        "run",
        "./cmd/bigclawctl",
        "automation",
        "e2e",
        "validation-bundle-policy-gate",
        *sys.argv[1:],
    ]
    return subprocess.call(command, cwd=go_root)


if __name__ == "__main__":
    raise SystemExit(main())
