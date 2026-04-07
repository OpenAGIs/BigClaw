# BIG-GO-1578 Python Asset Sweep

## Summary

`BIG-GO-1578` covered the following residual candidate set:

- `src/bigclaw/dashboard_run_contract.py`
- `src/bigclaw/memory.py`
- `src/bigclaw/repo_commits.py`
- `src/bigclaw/run_detail.py`
- `src/bigclaw/workspace_bootstrap_validation.py`
- `tests/test_dsl.py`
- `tests/test_live_shadow_scorecard.py`
- `tests/test_planning.py`
- `tests/test_reports.py`
- `tests/test_ui_review.py`
- `scripts/ops/symphony_workspace_validate.py`
- `bigclaw-go/scripts/e2e/external_store_validation.py`
- `bigclaw-go/scripts/migration/live_shadow_scorecard.py`

All candidate Python paths were already absent in the `main` baseline used for this lane, so the
sweep outcome is regression-hardening and exact replacement evidence rather than a new file delete.

## Counts

- Repository-wide physical Python file count before lane changes: `0`
- Repository-wide physical Python file count after lane changes: `0`
- Focused candidate set physical Python file count before lane changes: `0`
- Focused candidate set physical Python file count after lane changes: `0`
- Deleted files in this lane: `[]`
- Compatibility shims retained in this lane: `[]`

## Candidate Ledger

- `src/bigclaw/dashboard_run_contract.py` -> `bigclaw-go/internal/product/dashboard_run_contract.go`
- `src/bigclaw/memory.py` -> `bigclaw-go/internal/policy/memory.go`
- `src/bigclaw/repo_commits.py` -> `bigclaw-go/internal/collaboration/thread.go`
- `src/bigclaw/run_detail.py` -> `bigclaw-go/internal/observability/task_run.go`
- `src/bigclaw/workspace_bootstrap_validation.py` -> `bigclaw-go/internal/bootstrap/bootstrap.go`
- `tests/test_dsl.py` -> `bigclaw-go/internal/workflow/definition_test.go`
- `tests/test_live_shadow_scorecard.py` -> `bigclaw-go/cmd/bigclawctl/migration_commands.go`
- `tests/test_planning.py` -> `bigclaw-go/internal/planning/planning_test.go`
- `tests/test_reports.py` -> `bigclaw-go/internal/reporting/reporting_test.go`
- `tests/test_ui_review.py` -> `bigclaw-go/internal/uireview/uireview_test.go`
- `scripts/ops/symphony_workspace_validate.py` -> `scripts/ops/bigclawctl`
- `bigclaw-go/scripts/e2e/external_store_validation.py` -> `bigclaw-go/cmd/bigclawctl/automation_e2e_external_store_validation_command.go`
- `bigclaw-go/scripts/migration/live_shadow_scorecard.py` -> `bigclaw-go/cmd/bigclawctl/migration_commands.go`

## Evidence

- `src/bigclaw/dashboard_run_contract.py` stays retired; the Go-owned contract surface lives in
  `bigclaw-go/internal/product/dashboard_run_contract.go`.
- `src/bigclaw/memory.py` stays retired; the Go replacement tracked by earlier tranche coverage
  remains `bigclaw-go/internal/policy/memory.go`.
- `src/bigclaw/repo_commits.py` and `src/bigclaw/run_detail.py` stay retired; repo evidence and run
  detail ownership now live in `bigclaw-go/internal/collaboration/thread.go` and
  `bigclaw-go/internal/observability/task_run.go`.
- `src/bigclaw/workspace_bootstrap_validation.py` and
  `scripts/ops/symphony_workspace_validate.py` stay retired; supported validation entrypoints are
  `bigclaw-go/internal/bootstrap/bootstrap.go`, `scripts/ops/bigclawctl`, and
  `docs/go-cli-script-migration-plan.md`.
- `bigclaw-go/scripts/e2e/external_store_validation.py` stays retired; the supported command path is
  `bigclaw-go/cmd/bigclawctl/automation_e2e_external_store_validation_command.go`.
- `bigclaw-go/scripts/migration/live_shadow_scorecard.py` and its former Python test stay retired;
  the supported migration command and operator guidance now live in
  `bigclaw-go/cmd/bigclawctl/migration_commands.go` and `bigclaw-go/docs/migration-shadow.md`.

## Validation

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find src tests scripts/ops bigclaw-go/scripts/e2e bigclaw-go/scripts/migration -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1578(RepositoryHasNoPythonFiles|CandidatePathsStayAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepLedger)$'`

## Residual Risk

- This lane proves the listed candidates remain absent and that the documented Go/native owners
  still exist, but it does not add new functional behavior.
- `src/` and `tests/` are absent in the current baseline, so future regressions would show up as
  file reintroduction rather than diff drift inside those trees.
