#!/usr/bin/env python3
"""Legacy compatibility shim for the Go dev-smoke command."""

from __future__ import annotations

import subprocess
import sys
import warnings
from pathlib import Path

repo_root = Path(__file__).resolve().parents[1]

LEGACY_RUNTIME_GUIDANCE = (
    "bigclaw-go is the sole implementation mainline for active development; "
    "the legacy Python runtime surface remains migration-only."
)


def warn_legacy_runtime_surface(surface: str, replacement: str) -> str:
    message = f"{surface} is frozen for migration-only use. {LEGACY_RUNTIME_GUIDANCE} Use {replacement} instead."
    warnings.warn(message, DeprecationWarning, stacklevel=2)
    return message


def main() -> int:
    warn_legacy_runtime_surface("scripts/dev_smoke.py", "bash scripts/ops/bigclawctl dev-smoke")
    command = ["bash", str(repo_root / "scripts/ops/bigclawctl"), "dev-smoke", *sys.argv[1:]]
    return subprocess.call(command, cwd=repo_root)


if __name__ == "__main__":
    raise SystemExit(main())
