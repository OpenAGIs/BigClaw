#!/usr/bin/env python3
from __future__ import annotations

import sys
import subprocess
from pathlib import Path
from typing import Sequence

REPO_ROOT = Path(__file__).resolve().parents[2]
BIGCLAWCTL = REPO_ROOT / "scripts" / "ops" / "bigclawctl"


def build_command(argv: Sequence[str]) -> list[str]:
    command = ["bash", str(BIGCLAWCTL), "workspace", *argv]
    if not any(arg == "--repo-url" or arg.startswith("--repo-url=") for arg in argv):
        command.extend(["--repo-url", "git@github.com:OpenAGIs/BigClaw.git"])
    if not any(arg == "--cache-key" or arg.startswith("--cache-key=") for arg in argv):
        command.extend(["--cache-key", "openagis-bigclaw"])
    return command


def main(argv: Sequence[str] | None = None) -> int:
    if argv is None:
        argv = sys.argv[1:]
    completed = subprocess.run(build_command(argv), cwd=REPO_ROOT)
    return completed.returncode


if __name__ == "__main__":
    raise SystemExit(main())
