#!/usr/bin/env python3
"""Legacy compatibility shim for the Go refill command."""

from __future__ import annotations

import sys
from pathlib import Path

repo_root = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(repo_root / "src"))

from bigclaw.legacy_shim import run_bigclawctl_shim


def main() -> int:
    return run_bigclawctl_shim(__file__, ["refill"], sys.argv[1:])


if __name__ == "__main__":
    raise SystemExit(main())
