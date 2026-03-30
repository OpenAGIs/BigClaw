# BIG-GO-1006

## Plan
- Inventory the scoped Python batch: `tests/conftest.py`, `tests/test_control_center.py`, `tests/test_orchestration.py`, `tests/test_queue.py`, `tests/test_runtime_matrix.py`, `tests/test_scheduler.py`, `tests/test_workspace_bootstrap.py`.
- Map each scoped Python test to current repo-native Go coverage or identify why it must stay.
- Delete the scoped Python tests whose subject moved to Go or whose imported Python modules no longer exist.
- Keep `tests/conftest.py` only if the remaining Python suite still needs repo-local `src/` import bootstrapping.
- Replace stale Python validation/evidence references in `src/bigclaw/planning.py` and `tests/test_planning.py` with current Go-native coverage references.
- Measure repository-wide and batch-local Python file count deltas.
- Run targeted Go and Python tests for the touched planning and replacement coverage surfaces.
- Commit and push the branch changes for `BIG-GO-1006`.

## Batch Scope
- `tests/conftest.py`
- `tests/test_control_center.py`
- `tests/test_orchestration.py`
- `tests/test_queue.py`
- `tests/test_runtime_matrix.py`
- `tests/test_scheduler.py`
- `tests/test_workspace_bootstrap.py`

## Acceptance
- The batch file list is explicit and limited to the seven scoped files above.
- Python file count is reduced wherever the replaced behavior already has Go-native coverage.
- Each scoped file has a delete/replace/keep rationale captured in the final report.
- `src/bigclaw/planning.py` and `tests/test_planning.py` no longer point at removed Python runtime/foundation tests.
- The report includes repository-wide and batch-local Python file count impact.

## Validation
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1006/bigclaw-go && go test ./internal/queue ./internal/scheduler ./internal/bootstrap ./internal/workflow ./internal/worker ./internal/reporting ./internal/pilot ./internal/api`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1006 && PYTHONPATH=src python3 -m pytest tests/test_planning.py tests/test_operations.py tests/test_reports.py -q`
