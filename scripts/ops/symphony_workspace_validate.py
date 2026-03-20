#!/usr/bin/env python3
from __future__ import annotations

import sys
import subprocess
from pathlib import Path
from typing import Sequence

REPO_ROOT = Path(__file__).resolve().parents[2]
BIGCLAWCTL = REPO_ROOT / "scripts" / "ops" / "bigclawctl"


def build_command(argv: Sequence[str]) -> list[str]:
    return ["bash", str(BIGCLAWCTL), "workspace", "validate", *argv]


def main(argv: Sequence[str] | None = None) -> int:
    if argv is None:
        argv = sys.argv[1:]
    completed = subprocess.run(build_command(argv), cwd=REPO_ROOT)
    return completed.returncode


if __name__ == "__main__":
    raise SystemExit(main())
