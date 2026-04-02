# BIG-GO-1099

## Plan

- inline the legacy reports, operations, and evaluation surface from
  `src/bigclaw/reports.py` into `src/bigclaw/runtime.py`
- update package exports and compatibility shims so removed module imports keep
  resolving through `bigclaw.reports`, `bigclaw.operations`, and
  `bigclaw.evaluation`
- delete `src/bigclaw/reports.py`
- run targeted validation and capture the reduced `.py` count
- commit and push the branch

## Acceptance

- tracked repository `.py` count decreases from the current baseline of `4`
- `src/bigclaw/reports.py` is removed and its exported surface remains
  reachable from the package
- `src/bigclaw/runtime.py` owns the migrated report, operations, and evaluation
  legacy Python surface
- package-level compatibility for `bigclaw.reports`, `bigclaw.operations`, and
  `bigclaw.evaluation` remains intact

## Validation

- `python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/design_system.py src/bigclaw/runtime.py`
- `PYTHONPATH=src python3 - <<'PY' ...`
- `cd bigclaw-go && go test ./internal/evaluation ./internal/reporting ./internal/planning ./internal/regression ./internal/worker`
- `find src -name '*.py' | wc -l`
