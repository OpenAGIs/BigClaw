# BIG-GO-970 Workpad

## Plan

1. Inventory the Python files under `bigclaw-go/scripts/e2e/**` and `bigclaw-go/scripts/migration/**`, then map what each script does, how it is invoked, and whether a Go or shell-owned replacement already exists.
2. Keep the slice scoped to this issue by directly handling only those `e2e` and `migration` Python assets that can be deleted, replaced, or explicitly retained with a clear in-repo rationale.
3. Implement the minimal set of deletions or replacements that materially reduce Python file count in the target directories without widening into unrelated packages.
4. Record an explicit per-file disposition for the full batch: deleted, replaced, or retained, including the reason for each choice and the net Python-count change.
5. Run targeted validation commands for the touched scripts and packages, record exact commands and results, then commit and push the branch.

## Acceptance

- Produce the explicit Python file inventory for `bigclaw-go/scripts/e2e/**` and `bigclaw-go/scripts/migration/**`.
- Reduce the number of Python files in those directories as far as is safely possible within this issue.
- Document the delete / replace / retain decision for each Python file in scope.
- Report the impact on the overall Python file count.

## Validation

- Script-level validation for any new or modified non-Python replacements in `bigclaw-go/scripts/e2e/**` or `bigclaw-go/scripts/migration/**`.
- Targeted `go test` runs for any `bigclaw-go` Go packages introduced or touched by the replacements.
- `git status --short` to verify the final scoped change set before commit.

## Results

- Deleted 6 Python files from the scoped directories:
  - `bigclaw-go/scripts/e2e/run_task_smoke.py`
  - `bigclaw-go/scripts/e2e/broker_failover_stub_matrix_test.py`
  - `bigclaw-go/scripts/e2e/export_validation_bundle_test.py`
  - `bigclaw-go/scripts/e2e/multi_node_shared_queue_test.py`
  - `bigclaw-go/scripts/e2e/run_all_test.py`
  - `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.py`
- Repointed the live smoke wrappers and docs from the deleted Python shim to `go run ./cmd/bigclawctl automation e2e run-task-smoke`.
- Recorded the full retained/deleted/replaced inventory plus Python-count impact in `reports/BIG-GO-970-validation.md`.
- Python file count changed from `123` to `117` across the repository and from `19` to `13` inside `bigclaw-go/scripts/e2e/**` plus `bigclaw-go/scripts/migration/**`.

## Validation Results

- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-970/bigclaw-go && go test ./cmd/bigclawctl/... ./internal/regression/...` -> passed
  - `ok  	bigclaw-go/cmd/bigclawctl	2.147s`
  - `ok  	bigclaw-go/internal/regression	1.081s`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-970 && bash -n bigclaw-go/scripts/e2e/kubernetes_smoke.sh` -> passed
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-970 && bash -n bigclaw-go/scripts/e2e/ray_smoke.sh` -> passed
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-970 && bash -n bigclaw-go/scripts/e2e/run_all.sh` -> passed
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-970/bigclaw-go && go run ./cmd/bigclawctl automation e2e run-task-smoke --help` -> passed
