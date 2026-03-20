#!/usr/bin/env python3
from __future__ import annotations

import os
import sys
from pathlib import Path

sys.dont_write_bytecode = True

REPO_ROOT = Path(__file__).resolve().parents[2]
SRC_ROOT = REPO_ROOT / "src"
if str(SRC_ROOT) not in sys.path:
    sys.path.insert(0, str(SRC_ROOT))

from bigclaw.legacy_shim import LEGACY_PYTHON_WRAPPER_NOTICE, build_github_sync_args


if __name__ == "__main__":
    print(LEGACY_PYTHON_WRAPPER_NOTICE, file=sys.stderr)
    os.execvp("bash", build_github_sync_args(REPO_ROOT, sys.argv[1:]))
