#!/usr/bin/env python3
"""Legacy compatibility shim for the Go github-sync command."""

from __future__ import annotations

import sys

from bigclaw.legacy_shim import run_bigclawctl_shim


def main() -> int:
    return run_bigclawctl_shim(__file__, ["github-sync"], sys.argv[1:])


if __name__ == "__main__":
    raise SystemExit(main())
