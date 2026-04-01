# BIG-GO-1047 Workpad

## Plan
- Audit Python residues for workflow/runtime/scheduler/queue and identify direct in-repo consumers.
- Decouple remaining Python modules from legacy runtime shims where required to keep package imports coherent.
- Delete legacy Python implementation/test files for workflow/runtime/scheduler/queue surfaces and rely on Go mainline packages.
- Add or adjust Go tests only if coverage is needed for removed Python behavior.
- Run targeted validation, capture exact commands/results, verify Python file count decreases.
- Commit and push scoped changes to the remote branch.

## Acceptance
- Python files implementing or directly testing workflow/runtime/scheduler/queue legacy surfaces are removed.
- Remaining Python modules no longer require deleted legacy shim modules to import.
- Go workflow/scheduler/queue packages remain validated through targeted `go test` runs.
- Repository `find . -name "*.py" | wc -l` is lower than before the change.
- Commit message enumerates deleted Python files and added Go files/tests if any.

## Validation
- `find . -name "*.py" | wc -l`
- `cd bigclaw-go && go test ./internal/queue ./internal/scheduler ./internal/workflow`
- Package-level spot check for remaining Python imports if needed.

## Result
- Deleted Python implementation and test files:
  - `src/bigclaw/runtime.py`
  - `src/bigclaw/parallel_refill.py`
  - `scripts/ops/bigclaw_refill_queue.py`
  - `tests/test_control_center.py`
  - `tests/test_dsl.py`
  - `tests/test_evaluation.py`
  - `tests/test_orchestration.py`
  - `tests/test_queue.py`
  - `tests/test_risk.py`
  - `tests/test_runtime_matrix.py`
  - `tests/test_scheduler.py`
- Updated surviving Python and Go surfaces to stop referencing removed Python modules/tests.
- Pushed commits to `origin/BIG-GO-1047`:
  - `5782906f` `BIG-GO-1047 purge python workflow-runtime surfaces`
  - `76877ba4` `BIG-GO-1047 delete remaining python refill queue shims`
  - `3f5e4c33` `BIG-GO-1047 clean stale refs to removed python runtime files`
  - `989f835f` `BIG-GO-1047 align compile checks with removed python runtime files`
  - `d857cce3` `BIG-GO-1047 refresh deprecation contracts after python surface purge`
  - `febf56cb` `BIG-GO-1047 repair planning refs after python orchestration purge`

## Final Validation
- `find . -name "*.py" | wc -l` -> `50`
- `find src tests scripts/ops -name "*.py" | wc -l` -> `50`
- `python3 - <<'PY' ... import bigclaw ... PY` -> `ok bigclaw`
- `PYTHONPATH=src python3 -m pytest tests/test_planning.py tests/test_reports.py -q` -> `48 passed`
- `PYTHONPATH=src python3 -m pytest tests/test_planning.py tests/test_observability.py -q` -> `21 passed`
- `cd bigclaw-go && go test ./internal/queue ./internal/scheduler ./internal/workflow ./internal/refill` -> passed
- `cd bigclaw-go && go test ./internal/legacyshim ./cmd/bigclawctl` -> passed
- `cd bigclaw-go && go test ./internal/regression -run TestLegacyMainlineCompatibilityManifestStaysAligned` -> passed
