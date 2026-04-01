#!/usr/bin/env python3
"""Legacy compatibility shim for the Go workspace bootstrap command."""

from __future__ import annotations

import os
import subprocess
import sys
from pathlib import Path

def append_missing_flag(args: list[str], flag: str, value: str) -> list[str]:
    flag_prefix = flag + "="
    if any(arg == flag or arg.startswith(flag_prefix) for arg in args):
        return list(args)
    return [*args, flag, value]


def main() -> int:
    repo_root = Path(__file__).resolve().parents[2]
    args = append_missing_flag(
        sys.argv[1:],
        "--repo-url",
        os.getenv("BIGCLAW_BOOTSTRAP_REPO_URL", "git@github.com:OpenAGIs/BigClaw.git"),
    )
    args = append_missing_flag(args, "--cache-key", os.getenv("BIGCLAW_BOOTSTRAP_CACHE_KEY", "openagis-bigclaw"))
    command = ["bash", str(repo_root / "scripts/ops/bigclawctl"), "workspace", *args]
    return subprocess.call(command, cwd=repo_root)


if __name__ == "__main__":
    raise SystemExit(main())
