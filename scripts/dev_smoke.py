#!/usr/bin/env python3
"""Legacy compatibility shim for the Go dev-smoke command."""

from __future__ import annotations

import subprocess
import sys
from pathlib import Path

repo_root = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(repo_root / "src"))

from bigclaw.runtime import warn_legacy_runtime_surface


def main() -> int:
    warn_legacy_runtime_surface("scripts/dev_smoke.py", "bash scripts/ops/bigclawctl dev-smoke")
    command = ["bash", str(repo_root / "scripts/ops/bigclawctl"), "dev-smoke", *sys.argv[1:]]
    return subprocess.call(command, cwd=repo_root)


if __name__ == "__main__":
    raise SystemExit(main())
