# BIG-GO-990 Workpad

## Plan

1. Enumerate the current Python files under `bigclaw-go/scripts/e2e/**` and `bigclaw-go/scripts/migration/**`, plus the repository-wide Python file baseline.
2. Inspect each targeted script for live references and existing Go-native replacements.
3. Remove or replace Python entrypoints only where the Go path is already present and behaviorally adequate; keep the rest with explicit rationale.
4. Run targeted validation for touched scripts and any affected Go command surfaces.
5. Record exact commands/results, summarize Python-count impact, then commit and push the scoped change set.

## Acceptance

- Produce the exact `BIG-GO-990` batch file list for `bigclaw-go/scripts/e2e/**` and `bigclaw-go/scripts/migration/**`.
- Reduce Python files in the targeted directories where a repo-native Go replacement already exists.
- Document keep/replace/delete rationale for the batch.
- Report the impact on the repository-wide Python file count.

## Validation

- `rg --files -g '*.py' bigclaw-go/scripts/e2e bigclaw-go/scripts/migration`
- `rg --files -g '*.py' | wc -l`
- Targeted tests/commands for any touched scripts or replacement Go commands
- `git diff --check`
- `git status --short`

## Results

### Batch File List

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
- `bigclaw-go/scripts/migration/**`
  - No Python files were present in this worktree.

### Disposition

- Deleted `bigclaw-go/scripts/e2e/broker_failover_stub_matrix_test.py`.
  - Replaced by `bigclaw-go/scripts/e2e/broker_failover_stub_matrix_test.go`, which preserves coverage by invoking the Python generator from Go and asserting the same report/proof invariants.
- Deleted `bigclaw-go/scripts/e2e/export_validation_bundle_test.py`.
  - Replaced by `bigclaw-go/scripts/e2e/export_validation_bundle_test.go`, which preserves the prior render/index/broker/component coverage while removing a Python test file from the batch.
- Deleted `bigclaw-go/scripts/e2e/multi_node_shared_queue_test.py`.
  - Replaced by `bigclaw-go/scripts/e2e/multi_node_shared_queue_test.go`, which keeps the live takeover report assertions in Go.
- Deleted `bigclaw-go/scripts/e2e/run_all_test.py`.
  - Replaced by `bigclaw-go/scripts/e2e/run_all_test.go`, which preserves the shell harness and continuation gate coverage in Go.
- Kept `bigclaw-go/scripts/e2e/broker_failover_stub_matrix.py`.
  - Still owns the checked-in broker failover report surface and has no Go-native generator replacement.
- Kept `bigclaw-go/scripts/e2e/cross_process_coordination_surface.py`.
  - Still owns the checked-in coordination capability surface and has no Go-native generator replacement.
- Kept `bigclaw-go/scripts/e2e/export_validation_bundle.py`.
  - `run_all.sh` still shells through it; no Go-native bundle exporter exists yet.
- Kept `bigclaw-go/scripts/e2e/external_store_validation.py`.
  - Still owns the external store validation evidence pack and report schema.
- Kept `bigclaw-go/scripts/e2e/mixed_workload_matrix.py`.
  - Still generates the mixed workload matrix with no Go replacement.
- Kept `bigclaw-go/scripts/e2e/multi_node_shared_queue.py`.
  - Still owns the shared-queue and live takeover proof generation path.
- Kept `bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py`.
  - Still owns the deterministic takeover harness output with no Go replacement.

### Python Count Impact

- Repository Python files before: `105`
- Repository Python files after: `101`
- Targeted batch Python files before: `11`
- Targeted batch Python files after: `7`
- Net reduction in this lane: `4`

### Validation Record

- `cd bigclaw-go && go test ./scripts/e2e`
  - Result: `ok  	bigclaw-go/scripts/e2e	(cached)`
- `git diff --check`
  - Result: no whitespace or merge-marker errors.
- `rg --files -g '*.py' bigclaw-go/scripts/e2e bigclaw-go/scripts/migration | wc -l`
  - Result: `7`
- `rg --files -g '*.py' | wc -l`
  - Result: `101`
