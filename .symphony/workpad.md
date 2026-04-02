# BIG-GO-1099

## Plan

- inline the remaining legacy Python surface from `src/bigclaw/design_system.py`
  into `src/bigclaw/__init__.py`
- keep `bigclaw.design_system`, `bigclaw.runtime`, `bigclaw.reports`,
  `bigclaw.operations`, and `bigclaw.evaluation` import-compatible through
  package-root aliases and shims
- refresh Go regression, planning, and compile-check fixtures that still point
  at `src/bigclaw/design_system.py`
- delete `src/bigclaw/design_system.py`
- run targeted validation and capture the reduced `.py` count
- commit and push the branch

## Acceptance

- tracked repository `.py` count decreases from the current baseline of `2`
- `src/bigclaw/design_system.py` is removed and its exported surface remains
  reachable from the package
- `src/bigclaw/__init__.py` owns the remaining legacy Python surface
- package-level compatibility for `bigclaw.design_system`, `bigclaw.runtime`,
  `bigclaw.reports`, `bigclaw.operations`, and `bigclaw.evaluation` remains
  intact
- Go regression and compile-check fixtures no longer require
  `src/bigclaw/design_system.py` to exist

## Validation

- `python3 -m py_compile src/bigclaw/__init__.py`
- `PYTHONPATH=src python3 - <<'PY' ...`
- `cd bigclaw-go && go test ./internal/evaluation ./internal/reporting ./internal/planning ./internal/regression ./internal/worker ./internal/legacyshim`
- `find src -name '*.py' | wc -l`
