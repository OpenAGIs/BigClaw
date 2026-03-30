# BIG-GO-990 Workpad

## Scope

Target the remaining Python scripts under:

- `bigclaw-go/scripts/e2e/**`
- `bigclaw-go/scripts/migration/**`

Initial batch file list:

- `bigclaw-go/scripts/e2e/broker_failover_stub_matrix.py`
- `bigclaw-go/scripts/e2e/broker_failover_stub_matrix_test.py`
- `bigclaw-go/scripts/e2e/cross_process_coordination_surface.py`
- `bigclaw-go/scripts/e2e/export_validation_bundle.py`
- `bigclaw-go/scripts/e2e/export_validation_bundle_test.py`
- `bigclaw-go/scripts/e2e/external_store_validation.py`
- `bigclaw-go/scripts/e2e/mixed_workload_matrix.py`
- `bigclaw-go/scripts/e2e/multi_node_shared_queue.py`
- `bigclaw-go/scripts/e2e/multi_node_shared_queue_test.py`
- `bigclaw-go/scripts/e2e/run_all_test.py`
- `bigclaw-go/scripts/e2e/run_task_smoke.py`
- `bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.py`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py`
- `bigclaw-go/scripts/migration/export_live_shadow_bundle.py`
- `bigclaw-go/scripts/migration/live_shadow_scorecard.py`
- `bigclaw-go/scripts/migration/shadow_compare.py`
- `bigclaw-go/scripts/migration/shadow_matrix.py`

Current repository Python file count before this lane: `116`
Current targeted batch Python file count before this lane: `19`

## Plan

1. Inspect every Python file in the batch and map it to an existing Go/sh replacement or determine if a small Go port is needed.
2. Remove redundant Python files where a repository-native replacement already exists or add a Go-native replacement where missing and then remove the Python version.
3. Run targeted validation for the touched replacement paths.
4. Record exact file disposition, rationale, and repository Python count impact.
5. Commit and push the scoped changes for `BIG-GO-990`.

## Acceptance

- Produce the exact `BIG-GO-990` batch file list for `scripts/e2e` and `scripts/migration`.
- Reduce the number of Python files in the targeted directories as far as practical within this lane.
- Document keep/replace/delete rationale for every targeted Python file.
- Report the repository-wide Python file count impact.

## Validation

- `find . -name '*.py' | wc -l`
- Targeted validation commands for any Go/sh replacements touched in this lane
- `git status --short`
- `git log -1 --stat`

## Results

### File Disposition

- `bigclaw-go/scripts/e2e/run_task_smoke.py`
  - Deleted.
  - Reason: already fully replaced by `go run ./cmd/bigclawctl automation e2e run-task-smoke ...`; callers and docs in this lane now invoke the Go entrypoint directly.
- `bigclaw-go/scripts/migration/shadow_compare.py`
  - Deleted.
  - Reason: already fully replaced by `go run ./cmd/bigclawctl automation migration shadow-compare ...`; docs now point at the Go command directly.
- `bigclaw-go/scripts/migration/shadow_matrix.py`
  - Deleted.
  - Reason: replaced in this lane by `go run ./cmd/bigclawctl automation migration shadow-matrix ...`, with the matrix orchestration and corpus coverage logic moved into `cmd/bigclawctl`.
- `bigclaw-go/scripts/migration/live_shadow_scorecard.py`
  - Deleted.
  - Reason: replaced in this lane by `go run ./cmd/bigclawctl automation migration live-shadow-scorecard ...`, with the scorecard aggregation logic moved into `cmd/bigclawctl`.
- Remaining targeted Python files
  - Kept for now.
  - Reason: they still own report-generation or Python-only test behavior and do not yet have Go-native replacements in the repo.

### Python File Count Impact

- Repository Python files before: `116`
- Repository Python files after: `109`
- Targeted batch Python files before: `19`
- Targeted batch Python files after: `15`
- Net reduction: `4`

### Remaining Targeted Python Files

- `bigclaw-go/scripts/e2e/broker_failover_stub_matrix.py`
- `bigclaw-go/scripts/e2e/broker_failover_stub_matrix_test.py`
- `bigclaw-go/scripts/e2e/cross_process_coordination_surface.py`
- `bigclaw-go/scripts/e2e/export_validation_bundle.py`
- `bigclaw-go/scripts/e2e/export_validation_bundle_test.py`
- `bigclaw-go/scripts/e2e/external_store_validation.py`
- `bigclaw-go/scripts/e2e/mixed_workload_matrix.py`
- `bigclaw-go/scripts/e2e/multi_node_shared_queue.py`
- `bigclaw-go/scripts/e2e/multi_node_shared_queue_test.py`
- `bigclaw-go/scripts/e2e/run_all_test.py`
- `bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.py`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py`
- `bigclaw-go/scripts/migration/export_live_shadow_bundle.py`

### Validation Record

- `cd bigclaw-go && python3 -m unittest scripts/e2e/run_all_test.py`
  - Result: `Ran 3 tests in 4.250s` and `OK`
- `cd bigclaw-go && go test ./cmd/bigclawctl`
  - Result: `ok  	bigclaw-go/cmd/bigclawctl	3.623s`
- `cd bigclaw-go && python3 - <<'PY' ... PY`
  - Purpose: validate that `go run ./cmd/bigclawctl automation migration shadow-matrix ...` produces a matrix report with corpus coverage against stub HTTP endpoints.
  - Result: `shadow_matrix_cli_ok`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration --help`
  - Result: `usage: bigclawctl automation migration <shadow-compare|shadow-matrix|live-shadow-scorecard> [flags]`
- `cd bigclaw-go && python3 - <<'PY' ... PY`
  - Purpose: validate that `go run ./cmd/bigclawctl automation migration live-shadow-scorecard ...` emits a repo-native scorecard from compare/matrix JSON inputs.
  - Result: `live_shadow_scorecard_cli_ok`
- `find . -name '*.py' | wc -l`
  - Result: `109`
- `git status --short`
  - Result: only `.symphony/workpad.md` plus the scoped docs/script changes for this lane are modified.
