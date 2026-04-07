"""Minimal pytest bootstrap for frozen Python compatibility tests.

Removal condition: delete this file when the remaining Python tests are removed.
"""

import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SRC = ROOT / "src"
if str(SRC) not in sys.path:
    sys.path.insert(0, str(SRC))
