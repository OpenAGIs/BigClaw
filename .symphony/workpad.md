# BIG-GO-965 Workpad

## Plan

1. Inventory the Python tests directly in scope for `tests/conftest.py` plus the runtime/service/scheduler/workflow/orchestration/queue lane, and record the exact file list for this issue.
2. Consolidate the runtime/scheduler/workflow/orchestration/queue pytest assets into fewer Python files without changing the covered behaviors.
3. Update any code or tests that still reference the removed file paths so the compressed test layout remains internally consistent.
4. Run targeted pytest validation for the consolidated lane, compare Python file counts before and after, then commit and push the scoped branch changes.

## Acceptance

- Provide the explicit Python file list directly handled by `BIG-GO-965`.
- Reduce the number of Python files in the targeted pytest lane.
- Record delete/replace/keep reasons for each directly handled Python file.
- Report the impact on the repository-wide Python file count.

## Validation

- `PYTHONPATH=src python3 -m pytest` for the consolidated runtime lane test file.
- `python3 -m pytest tests/test_planning.py -q` if planning/test path references are touched.
- `git diff --stat` and `find . -name '*.py' | wc -l` to confirm scoped changes and file-count impact.

## Results

- Direct file list for this issue:
  - `tests/conftest.py`
  - `tests/test_runtime.py`
  - `tests/test_scheduler.py`
  - `tests/test_workflow.py`
  - `tests/test_orchestration.py`
  - `tests/test_queue.py`
  - `tests/test_runtime_core.py`
- Handling decision by file:
  - `tests/conftest.py`: kept; it is still the narrowest shared pytest bootstrap for `src/` imports and removing it would widen scope across the whole test suite.
  - `tests/test_runtime.py`: deleted and replaced by `tests/test_runtime_core.py`; runtime assertions remain unchanged but no longer need a dedicated file.
  - `tests/test_scheduler.py`: deleted and replaced by `tests/test_runtime_core.py`; scheduler routing and budget coverage stay in the merged core lane.
  - `tests/test_workflow.py`: deleted and replaced by `tests/test_runtime_core.py`; workflow acceptance and closeout coverage remains in the merged file.
  - `tests/test_orchestration.py`: deleted and replaced by `tests/test_runtime_core.py`; orchestration policy and handoff coverage remains in the merged file.
  - `tests/test_queue.py`: deleted and replaced by `tests/test_runtime_core.py`; queue persistence coverage remains in the merged file, including the embedded validation path string update.
  - `tests/test_runtime_core.py`: added as the single replacement test asset for the runtime/service/scheduler/workflow/orchestration/queue pytest foundation/core lane.
- Related path-reference updates:
  - `src/bigclaw/planning.py`
  - `tests/test_planning.py`
- Python file count impact:
  - Before: `123`
  - After: `119`
  - Net change: `-4` Python files repository-wide.
- Validation results:
  - `python3 -m py_compile tests/test_runtime_core.py src/bigclaw/planning.py tests/test_planning.py tests/conftest.py` -> passed
  - `PYTHONPATH=src python3 -m pytest tests/test_runtime_core.py -q` -> `26 passed in 0.15s`
  - `PYTHONPATH=src python3 -m pytest tests/test_planning.py -q` -> `14 passed in 0.13s`
