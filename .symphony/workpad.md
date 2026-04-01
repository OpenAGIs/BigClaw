# BIG-GO-1064 Workpad

## Plan
- Capture the pre-change Python file baseline for the repo and confirm which suggested residual tests still exist in this workspace.
- Inspect the remaining Python test assets in scope: `tests/conftest.py`, `tests/test_console_ia.py`, `tests/test_control_center.py`, `tests/test_design_system.py`, and `tests/test_evaluation.py`.
- Map each in-scope Python test to an existing Go replacement, or decide that the Python file can be deleted with no replacement because Go regression coverage already pins the surface.
- Remove the in-scope Python test assets that are now redundant and add or adjust Go regression coverage only where the deletion would otherwise become unverifiable.
- Run targeted validation, record exact commands and outcomes, then commit and push the branch.

## Acceptance
- This batch explicitly accounts for the in-scope residual Python assets that still exist in the workspace.
- The chosen Python test files are deleted, replaced, or otherwise downgraded away from live Python execution.
- Go-side validation remains available for the affected product surfaces after the Python removals.
- Exact validation commands and outcomes are captured for closeout.
- Repo-wide physical Python file count decreases from the pre-change baseline.

## Validation
- `find . -type f \\( -name '*.py' -o -name '*.pyi' \\) | sort | wc -l`
- `find tests -type f \\( -name '*.py' -o -name '*.pyi' \\) | sort`
- `cd bigclaw-go && go test ./internal/product ./internal/regression`
- `git status --short`

## Scope Results
- Pre-change repo Python file count at `HEAD`: `43`
- Post-change repo Python file count in workspace: `39`
- Net reduction this tranche: `4` Python files
- Pre-change in-scope `tests/` Python assets: `tests/conftest.py`, `tests/test_console_ia.py`, `tests/test_control_center.py`, `tests/test_design_system.py`, `tests/test_evaluation.py`
- Removed in this tranche: `tests/test_console_ia.py`, `tests/test_control_center.py`, `tests/test_design_system.py`, `tests/test_evaluation.py`
- Retained in scope: `tests/conftest.py` because the repo still has active pytest coverage under `tests/` and this shared fixture file remains live infrastructure rather than a product-surface test asset.

## Validation Results
- `find . -type f \( -name '*.py' -o -name '*.pyi' \) | sort | wc -l`
  Result: `39`
- `find tests -type f \( -name '*.py' -o -name '*.pyi' \) | sort`
  Result: `tests/conftest.py`, `tests/test_live_shadow_bundle.py`, `tests/test_models.py`, `tests/test_observability.py`, `tests/test_operations.py`, `tests/test_orchestration.py`, `tests/test_planning.py`, `tests/test_queue.py`, `tests/test_repo_collaboration.py`, `tests/test_repo_links.py`, `tests/test_repo_rollout.py`, `tests/test_reports.py`, `tests/test_risk.py`, `tests/test_runtime_matrix.py`, `tests/test_scheduler.py`, `tests/test_ui_review.py`
- `PYTHONPATH=src python3 -m pytest tests/test_planning.py -q`
  Result: `14 passed in 0.09s`
- `cd bigclaw-go && go test ./internal/product ./internal/regression`
  Result: `ok   bigclaw-go/internal/product 1.645s`; `ok   bigclaw-go/internal/regression 1.963s`

## Residual Risk
- `src/bigclaw/planning.py` still emits Python-side candidate metadata; this tranche only repointed its validation/evidence references away from deleted Python tests and did not migrate the planner module itself.
- `tests/conftest.py` remains until a broader pytest retirement tranche removes the remaining Python test suite or replaces its shared fixture behavior.
