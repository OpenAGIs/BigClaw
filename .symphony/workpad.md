# BIG-GO-962 Workpad

## Scope

Targeted legacy Python modules under `src/bigclaw` for this lane:

- `src/bigclaw/runtime.py`
- `src/bigclaw/service.py`
- `src/bigclaw/scheduler.py`
- `src/bigclaw/workflow.py`
- `src/bigclaw/orchestration.py`
- `src/bigclaw/queue.py`

Current repository Python file count before this lane: `50`

## Plan

1. Confirm the exact lane-owned Python files and their import/test dependencies.
2. Consolidate the six targeted modules into fewer implementation files inside `src/bigclaw` while preserving the legacy import surface used by package code and tests.
3. Delete the superseded module files once compatibility aliases are in place.
4. Run targeted validation for the touched legacy runtime/service workflow surfaces and record exact commands and results here.
5. Report the direct file list, deletion/replacement/retention rationale, and the net impact on total Python file count.
6. Commit and push the scoped lane changes.

## Acceptance

- Produce the exact Python file list directly owned by `BIG-GO-962`.
- Reduce the number of Python files in the targeted runtime/service/scheduler/workflow/orchestration/queue surface.
- Preserve the import-compatible legacy API for `bigclaw.runtime`, `bigclaw.service`, `bigclaw.scheduler`, `bigclaw.workflow`, `bigclaw.orchestration`, and `bigclaw.queue`.
- Record delete/replace/retain reasoning for each targeted legacy file.
- Report before/after total Python file counts for the repository.

## Validation

- Import smoke checks for the legacy module names after consolidation.
- Targeted test execution for affected legacy surfaces, using the project-supported Python test runner available in this checkout.
- `git status --short` to confirm the change set stays scoped to this lane.

## Notes

- The target scope in this checkout is six top-level modules, not nested directories.
- `pytest` is not currently on `PATH`; validation must use the project’s available Python environment tooling.

## Results

### File Disposition

- `src/bigclaw/runtime.py`
  - Retained and expanded.
  - Reason: became the single implementation home for runtime, service, queue, orchestration, scheduler, and workflow compatibility surfaces.
- `src/bigclaw/service.py`
  - Deleted.
  - Reason: implementation moved into `src/bigclaw/runtime.py`; `bigclaw.service` import compatibility is now provided from `src/bigclaw/__init__.py`, including the `python -m bigclaw serve` CLI path.
- `src/bigclaw/queue.py`
  - Deleted.
  - Reason: implementation moved into `src/bigclaw/runtime.py`; `bigclaw.queue` import compatibility is now provided from `src/bigclaw/__init__.py`.
- `src/bigclaw/orchestration.py`
  - Deleted.
  - Reason: implementation moved into `src/bigclaw/runtime.py`; `bigclaw.orchestration` import compatibility is now provided from `src/bigclaw/__init__.py`.
- `src/bigclaw/scheduler.py`
  - Deleted.
  - Reason: implementation moved into `src/bigclaw/runtime.py`; `bigclaw.scheduler` import compatibility is now provided from `src/bigclaw/__init__.py`.
- `src/bigclaw/workflow.py`
  - Deleted.
  - Reason: implementation moved into `src/bigclaw/runtime.py`; `bigclaw.workflow` import compatibility is now provided from `src/bigclaw/__init__.py`.

### Python File Count Impact

- Repository `src/bigclaw` Python files before: `50`
- Repository `src/bigclaw` Python files after: `45`
- Net reduction: `5`

### Validation Record

- `python3 -m compileall src/bigclaw`
  - Result: success
- `PYTHONPATH=src python3 - <<'PY' ...`
  - Result: success; verified `bigclaw.runtime`, `bigclaw.service`, `bigclaw.queue`, `bigclaw.orchestration`, `bigclaw.scheduler`, and `bigclaw.workflow` all import cleanly.
- `PYTHONPATH=src python3 -m bigclaw --help`
  - Result: success; verified the CLI still resolves `from .service import run_server` through the compatibility shim.
- `PYTHONPATH=src python3 -m pytest tests/test_runtime.py tests/test_runtime_matrix.py tests/test_scheduler.py tests/test_orchestration.py tests/test_queue.py tests/test_workflow.py tests/test_control_center.py tests/test_execution_flow.py tests/test_dsl.py tests/test_evaluation.py tests/test_risk.py tests/test_audit_events.py tests/test_operations.py tests/test_reports.py`
  - Result: `107 passed in 0.18s`
