#!/usr/bin/env python3
"""Legacy compatibility shim for the Go workspace bootstrap command."""

from __future__ import annotations

import os
import subprocess
import sys

from bigclaw.legacy_shim import append_missing_flag, repo_root_from_script


def main() -> int:
    repo_root = repo_root_from_script(__file__)
    args = append_missing_flag(sys.argv[1:], "--repo-url", os.getenv("BIGCLAW_BOOTSTRAP_REPO_URL", "git@github.com:OpenAGIs/BigClaw.git"))
    args = append_missing_flag(args, "--cache-key", os.getenv("BIGCLAW_BOOTSTRAP_CACHE_KEY", "openagis-bigclaw"))
    command = ["bash", str(repo_root / "scripts/ops/bigclawctl"), "workspace", *args]
    return subprocess.call(command, cwd=repo_root)


if __name__ == "__main__":
    raise SystemExit(main())
