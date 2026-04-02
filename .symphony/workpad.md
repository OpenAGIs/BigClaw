# BIG-GO-1099

## Plan

- inline the remaining legacy runtime, workflow, reports, operations, and
  evaluation surface from `src/bigclaw/runtime.py` into
  `src/bigclaw/design_system.py`
- update package exports and compatibility shims so removed module imports keep
  resolving through `bigclaw.runtime`, `bigclaw.reports`, `bigclaw.operations`,
  and `bigclaw.evaluation`
- refresh Go regression, planning, and compile-check fixtures that still point
  at `src/bigclaw/runtime.py`
- delete `src/bigclaw/runtime.py`
- run targeted validation and capture the reduced `.py` count
- commit and push the branch

## Acceptance

- tracked repository `.py` count decreases from the current baseline of `3`
- `src/bigclaw/runtime.py` is removed and its exported surface remains
  reachable from the package
- `src/bigclaw/design_system.py` owns the remaining legacy Python surface
- package-level compatibility for `bigclaw.runtime`, `bigclaw.reports`,
  `bigclaw.operations`, and `bigclaw.evaluation` remains intact
- Go regression and compile-check fixtures no longer require
  `src/bigclaw/runtime.py` to exist

## Validation

- `python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/design_system.py`
- `PYTHONPATH=src python3 - <<'PY' ...`
- `cd bigclaw-go && go test ./internal/evaluation ./internal/reporting ./internal/planning ./internal/regression ./internal/worker ./internal/legacyshim`
- `find src -name '*.py' | wc -l`
