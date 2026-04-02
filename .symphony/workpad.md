# BIG-GO-964 Workpad

## Plan

1. Inventory the remaining `src/bigclaw/*.py` modules and lock this batch to the final removable host candidate: `design_system.py`.
2. Fold the `bigclaw.design_system` implementation into `execution_contract.py`, which is already the surviving shared contract host for non-runtime dataclasses and report renderers.
3. Update `src/bigclaw/__init__.py` to import the former design-system surface from `execution_contract.py` and register `bigclaw.design_system` as a legacy module alias before `console_ia` / `ui_review`.
4. Delete `src/bigclaw/design_system.py`, then run targeted import and pytest validation for the touched surfaces.
5. Recount Python files under `src/bigclaw` and across the repo, record exact command results, then commit and push the scoped change set.

## Acceptance

- Explicit handled-file list for this batch:
  - `src/bigclaw/design_system.py`
  - `src/bigclaw/execution_contract.py`
  - `src/bigclaw/__init__.py`
- Reduce the number of Python files under `src/bigclaw/` in this batch.
- Document delete / replace / retain reasons for the handled files.
- Report the before/after impact on both `src/bigclaw/*.py` and total repo `*.py` counts.

## Validation

- `PYTHONPATH=src python3 - <<'PY'` import check for:
  - `bigclaw.design_system`
  - `bigclaw.console_ia`
  - `bigclaw.ui_review`
- `PYTHONPATH=src python3 -m pytest tests/test_design_system.py tests/test_console_ia.py tests/test_ui_review.py tests/test_execution_contract.py`
- `python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/__main__.py src/bigclaw/execution_contract.py src/bigclaw/scheduler.py`
- `find src/bigclaw -type f -name '*.py' | wc -l`
- `find . -type f -name '*.py' | wc -l`
- `git status --short`

## Baseline

- `find src/bigclaw -type f -name '*.py' | wc -l`
  - Result: `5`
- `find . -type f -name '*.py' | wc -l`
  - Result: `79`

## Decisions

- `src/bigclaw/design_system.py`
  - Delete. Its implementation was folded into `src/bigclaw/execution_contract.py` because the remaining direct consumers were only external import surfaces, not internal runtime dependencies.
- `src/bigclaw/execution_contract.py`
  - Replace/expand. It now hosts both the shared execution contracts and the former design-system/UI-review datamodel and report helpers.
- `src/bigclaw/__init__.py`
  - Retain. It remains the package compatibility layer and now aliases `bigclaw.design_system` to `bigclaw.execution_contract` before `console_ia` and `ui_review`.

## Results

- `PYTHONPATH=src python3 - <<'PY'` import check
  - Result:
    - `bigclaw.design_system -> bigclaw.execution_contract`
    - `bigclaw.console_ia -> bigclaw.execution_contract`
    - `bigclaw.ui_review -> bigclaw.execution_contract`
- `PYTHONPATH=src python3 -m pytest tests/test_design_system.py tests/test_console_ia.py tests/test_ui_review.py tests/test_execution_contract.py`
  - Result: `60 passed in 0.11s`
- `python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/__main__.py src/bigclaw/execution_contract.py src/bigclaw/scheduler.py`
  - Result: passed with no output
- `find src/bigclaw -type f -name '*.py' | wc -l`
  - Result: `4`
- `find . -type f -name '*.py' | wc -l`
  - Result: `78`
- `git status --short`
  - Result:
    - `M .symphony/workpad.md`
    - `M src/bigclaw/__init__.py`
    - `D src/bigclaw/design_system.py`
    - `M src/bigclaw/execution_contract.py`

## Continuation Assessment

- Remaining `src/bigclaw/*.py` floor after this batch:
  - `src/bigclaw/__init__.py`
  - `src/bigclaw/__main__.py`
  - `src/bigclaw/execution_contract.py`
  - `src/bigclaw/scheduler.py`
- Retention rationale for the remaining floor:
  - `src/bigclaw/__init__.py`
    - Retain. It is the only place registering legacy module aliases such as `bigclaw.design_system`, `bigclaw.console_ia`, and `bigclaw.ui_review`; deleting it would break the migration compatibility import surface.
  - `src/bigclaw/__main__.py`
    - Retain. `python -m bigclaw` requires a physical `__main__.py`; moving its contents elsewhere would not reduce file count because the entrypoint shim must still exist.
  - `src/bigclaw/execution_contract.py`
    - Retain. It is now the consolidated shared-contract and UI-contract host; removing it would force a wider re-split rather than another compression step.
  - `src/bigclaw/scheduler.py`
    - Retain. It remains the consolidated runtime/orchestration host used directly by runtime-focused tests and by `__main__.py`.
- Additional evidence:
  - `rg -n 'from bigclaw import|import bigclaw($| )' src tests`
    - Result: no matches
  - No package-root imports remain that would justify expanding `__init__.py` into a full implementation host.
  - `__main__.py` and `scheduler.py` still have direct runtime coverage and are not compatibility-only shells.

## Continuation Validation

- `PYTHONPATH=src python3 -m pytest tests/test_scheduler.py tests/test_runtime.py tests/test_execution_flow.py tests/test_orchestration.py`
  - Result: `16 passed in 0.08s`
- `PYTHONPATH=src python3 -m pytest tests/test_github_sync.py tests/test_workspace_bootstrap.py`
  - Result: `14 passed in 3.99s`
