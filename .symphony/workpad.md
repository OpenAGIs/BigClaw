# BIG-GO-1040 Workpad

## Plan

1. Inventory remaining repository Python files and select a scoped tranche whose behavior is already covered in `bigclaw-go/` or can be replaced with small Go tests in the same domain package.
2. Delete the selected Python tests and add or extend Go tests so the replaced behavior still has checked-in coverage.
3. Verify the repo state with targeted `go test` commands plus file-count checks for remaining `.py` files and absent Python packaging files.
4. Commit the scoped migration change on an issue branch and push it to the remote.

## Acceptance

- The number of repository `.py` files decreases within this issue scope.
- No new Python files are introduced.
- Every deleted Python test is covered by an existing or newly added Go test in `bigclaw-go/`.
- `pyproject.toml` and `setup.py` remain absent.
- The final change can name which Python files were removed and which Go test files were added or expanded.

## Validation

- `find . -name '*.py' | sort | wc -l`
- `find tests -maxdepth 1 -name '*.py' | sort`
- `find . \\( -name pyproject.toml -o -name setup.py \\) -print | sort`
- Targeted `go test` commands for each touched Go package
- `git status --short`

## Validation Results

- `find tests -maxdepth 1 -name '*.py' | sort`
  - `tests/conftest.py`
  - `tests/test_console_ia.py`
  - `tests/test_control_center.py`
  - `tests/test_design_system.py`
  - `tests/test_dsl.py`
  - `tests/test_evaluation.py`
  - `tests/test_event_bus.py`
  - `tests/test_live_shadow_bundle.py`
  - `tests/test_memory.py`
  - `tests/test_models.py`
  - `tests/test_observability.py`
  - `tests/test_operations.py`
  - `tests/test_orchestration.py`
  - `tests/test_parallel_validation_bundle.py`
  - `tests/test_planning.py`
  - `tests/test_queue.py`
  - `tests/test_repo_collaboration.py`
  - `tests/test_repo_links.py`
  - `tests/test_repo_rollout.py`
  - `tests/test_repo_triage.py`
  - `tests/test_reports.py`
  - `tests/test_risk.py`
  - `tests/test_runtime_matrix.py`
  - `tests/test_scheduler.py`
  - `tests/test_ui_review.py`
  - `tests/test_validation_bundle_continuation_policy_gate.py`
  - `tests/test_validation_policy.py`
- `find tests -maxdepth 1 -name '*.py' | sort | wc -l`
  - `27`
- `find . -name '*.py' | sort | wc -l`
  - `77`
- `find . \( -name pyproject.toml -o -name setup.py \) -print | sort`
  - no output
- `cd bigclaw-go && go test ./internal/repo`
  - `ok  	bigclaw-go/internal/repo	0.432s`
