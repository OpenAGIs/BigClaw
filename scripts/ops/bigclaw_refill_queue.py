#!/usr/bin/env python3
"""Legacy compatibility shim for the Go refill command."""

from __future__ import annotations

import subprocess
import sys
from pathlib import Path

def repo_root_from_script(script_path: str) -> Path:
    return Path(script_path).resolve().parents[2]


def run_bigclawctl_shim(script_path: str, command: list[str], forwarded: list[str]) -> int:
    repo_root = repo_root_from_script(script_path)
    argv = ["bash", str(repo_root / "scripts/ops/bigclawctl"), *command, *forwarded]
    return subprocess.call(argv, cwd=repo_root)


def main() -> int:
    return run_bigclawctl_shim(__file__, ["refill"], list(sys.argv[1:]))


if __name__ == "__main__":
    raise SystemExit(main())
