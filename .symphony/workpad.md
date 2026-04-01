# BIG-GO-1043 Workpad

## Plan

1. Replace the first repo-focused Python test tranche under `tests/` with Go-native tests in `bigclaw-go/internal/repo`.
2. Extend `bigclaw-go/internal/repo/repo_surfaces_test.go` for any assertions still only present in Python.
3. Delete the replaced Python test files from `tests/`.
4. Run targeted `go test` coverage for `bigclaw-go/internal/repo` and record exact commands and results.
5. Verify the top-level Python test count drops, then commit and push the scoped change set.

## Acceptance

- Deleted Python files are limited to this tranche:
  - `tests/test_repo_board.py`
  - `tests/test_repo_gateway.py`
  - `tests/test_repo_governance.py`
  - `tests/test_repo_links.py`
  - `tests/test_repo_registry.py`
  - `tests/test_repo_triage.py`
- The deleted Python coverage is replaced by checked-in Go tests in `bigclaw-go/internal/repo/repo_surfaces_test.go` and existing repo Go tests remain green.
- No new Python files are added.
- `find . -name '*.py' | wc -l` decreases from the starting count for this workspace.
- Final delivery lists deleted Python files and the Go test file updated as the replacement.

## Validation

- `cd bigclaw-go && go test ./internal/repo`
- `find tests -maxdepth 1 -type f -name '*.py' | sort`
- `find . -name '*.py' | wc -l`
- `git status --short`

## Validation Results

- `cd bigclaw-go && go test ./internal/repo`
  - `ok  	bigclaw-go/internal/repo	0.885s`
- `find tests -maxdepth 1 -type f -name '*.py' | sort`
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
  - `tests/test_repo_rollout.py`
  - `tests/test_reports.py`
  - `tests/test_risk.py`
  - `tests/test_runtime_matrix.py`
  - `tests/test_scheduler.py`
  - `tests/test_ui_review.py`
  - `tests/test_validation_bundle_continuation_policy_gate.py`
  - `tests/test_validation_policy.py`
- `find . -name '*.py' | wc -l`
  - `75`
- `git status --short`
  - `M .symphony/workpad.md`
  - `M bigclaw-go/internal/repo/board.go`
  - `M bigclaw-go/internal/repo/repo_surfaces_test.go`
  - `D tests/test_repo_board.py`
  - `D tests/test_repo_gateway.py`
  - `D tests/test_repo_governance.py`
  - `D tests/test_repo_links.py`
  - `D tests/test_repo_registry.py`
  - `D tests/test_repo_triage.py`
