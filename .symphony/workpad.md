# BIG-GO-926

## Plan
- Inventory the current Python and non-Go assets for `tests/test_reports.py`, `tests/test_validation*.py`, and `tests/test_evaluation.py`, then map each area to an existing or new Go package under `bigclaw-go`.
- Implement the first Go replacements in the smallest scoped packages that cover reporting, evaluation, and validation policy behavior needed by the migrated tests.
- Add Go tests that preserve the relevant behavioral coverage from the Python suite, remove migrated Python tests/assets when safe, and keep any untouched legacy assets explicit in the inventory.
- Run targeted regression commands for the new Go tests and any impacted existing tests, then commit and push the branch.

## Acceptance
- Current Python and non-Go assets in scope are explicitly listed in repo changes.
- Go replacement implementation or migration plan exists for reporting, evaluation, and validation policy coverage.
- First batch of Go implementation and migrated tests lands in this branch.
- Conditions for deleting old Python assets and exact regression commands are documented.

## Asset Inventory
- Deleted root Python test entrypoints now covered by Go:
  - `tests/test_reports.py`
  - `tests/test_evaluation.py`
  - `tests/test_validation_policy.py`
  - `tests/test_validation_bundle_continuation_policy_gate.py`
  - `tests/test_validation_bundle_continuation_scorecard.py`
- Deleted isolated Python source now superseded by Go:
  - `src/bigclaw/validation_policy.py`
  - `src/bigclaw/evaluation.py`
- Deleted unused legacy report helpers and exports from `src/bigclaw/reports.py` / `src/bigclaw/__init__.py`:
  - issue validation / closure helpers
  - report studio types and bundle writers
  - launch/final delivery checklist types and renderers
  - pilot portfolio helper
- Deleted Python sibling test now superseded by Go regression coverage:
  - `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.py`
- New Go replacement coverage:
  - `bigclaw-go/internal/reporting/migration_suite.go`
  - `bigclaw-go/internal/reporting/migration_suite_test.go`
  - `bigclaw-go/internal/regression/validation_bundle_continuation_migration_test.go`
- New Python helper split landed to shrink the remaining legacy report surface:
  - `src/bigclaw/reporting_common.py`
- Remaining Python / non-Go assets still in scope:
  - `src/bigclaw/reports.py`
  - `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py`
  - `bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py`
  - checked-in report fixtures under `bigclaw-go/docs/reports/validation-bundle-continuation*.json` and dependent live-validation docs

## Remaining Migration Plan
- `reports` / `evaluation` / `validation_policy` root Python test coverage is migrated into `bigclaw-go/internal/reporting`.
- continuation script wrapper coverage is migrated into Go regression tests under `bigclaw-go/internal/regression`, while the Python scripts themselves still remain as runtime assets.
- Current concrete blockers to deleting the remaining Python sources:
  - `src/bigclaw/reports.py` is still imported by `src/bigclaw/__main__.py`, `src/bigclaw/workflow.py`, `src/bigclaw/scheduler.py`, `src/bigclaw/__init__.py`, and Python tests including `tests/test_repo_rollout.py`, `tests/test_workflow.py`, `tests/test_audit_events.py`, and `tests/test_observability.py`.
  - `src/bigclaw/reporting_common.py` is still imported by `src/bigclaw/__main__.py`, `src/bigclaw/workflow.py`, `src/bigclaw/scheduler.py`, `src/bigclaw/operations.py`, `src/bigclaw/__init__.py`, and Python tests including `tests/test_control_center.py` and `tests/test_operations.py`.
- Remaining external `src/bigclaw/reports.py` symbol usage after this issue:
  - `src/bigclaw/__main__.py`: `render_repo_sync_audit_report`
  - `src/bigclaw/workflow.py`: `build_orchestration_canvas`, `PilotScorecard`, `render_orchestration_canvas`, `render_pilot_scorecard`, `render_repo_sync_audit_report`
  - `src/bigclaw/scheduler.py`: `render_task_run_detail_page`, `render_task_run_report`
  - `tests/test_repo_rollout.py`: `render_repo_narrative_exports`, `render_weekly_repo_evidence_section`
  - `tests/test_workflow.py`: `PilotMetric`, `PilotScorecard`
  - `tests/test_audit_events.py`: `build_orchestration_canvas_from_ledger_entry`, `build_takeover_queue_from_ledger`
  - `tests/test_observability.py`: `render_repo_sync_audit_report`, `render_task_run_detail_page`, `render_task_run_report`
- Remaining external `src/bigclaw/reporting_common.py` symbol usage after this issue:
  - `src/bigclaw/__main__.py`: `write_report`
  - `src/bigclaw/workflow.py`: `write_report`
  - `src/bigclaw/scheduler.py`: `write_report`
  - `src/bigclaw/operations.py`: `SharedViewContext`, `build_console_actions`, `render_console_actions`, `render_shared_view_context`, `write_report`
  - `tests/test_control_center.py`: `SharedViewContext`, `SharedViewFilter`
  - `tests/test_operations.py`: `SharedViewContext`, `SharedViewFilter`
- Safe deletion conditions for remaining Python assets:
  - delete `src/bigclaw/reports.py` only after its remaining importers under `tests/` and `src/bigclaw/*` are migrated or removed;
  - delete `src/bigclaw/reporting_common.py` only after the remaining legacy Python runtime and tests stop importing shared view / report-writing helpers;
  - delete `bigclaw-go/scripts/e2e/*.py` only after the scripts themselves are replaced by Go executables/tests and docs/runbooks stop invoking Python.

## Validation
- `cd bigclaw-go && go test ./internal/reporting -count=1`
  - result: `ok  	bigclaw-go/internal/reporting	0.805s`
- `cd bigclaw-go && go test ./internal/regression -run 'TestValidationBundleContinuation(ScorecardCheckedInShape|ScorecardScriptBuildReport|PolicyGatePartialLaneHistoryHold|PolicyGateCanAllowPartialLaneHistory|PolicyGateCheckedInShape|PolicyGateCLIReturnsZeroForCheckedInGo)$' -count=1`
  - result: `ok  	bigclaw-go/internal/regression	0.812s`
- `python3 -m pytest tests/test_operations.py -q`
  - result: `20 passed`
- `python3 -m pytest tests/test_workflow.py -q`
  - result: `8 passed`
- `python3 -m pytest tests/test_workflow.py tests/test_operations.py tests/test_repo_rollout.py tests/test_control_center.py tests/test_audit_events.py tests/test_observability.py -q`
  - result: `45 passed`
- `python3 -m pytest tests/test_control_center.py tests/test_operations.py tests/test_workflow.py tests/test_observability.py -q`
  - result: `38 passed`
- `python3 -m pytest tests/test_repo_rollout.py tests/test_audit_events.py -q`
  - result: `7 passed`
- `python3 -c "import sys; sys.path.insert(0, 'src'); import bigclaw; print('ok')"`
  - result: `ok`
- `git status --short`
