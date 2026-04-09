# BIG-GO-194 Workpad

## Plan

1. Inventory Python-named scripts, wrappers, and CLI helpers that already have non-Python behavior or Go-backed replacements.
2. Replace residual Python entrypoints in scope with non-Python paths, keeping compatibility where practical through shell wrappers.
3. Update repo references and packaging/lint configuration so the renamed helpers are the documented canonical surface.
4. Run targeted validation for the touched wrappers and helper commands, then record exact commands and outcomes.
5. Commit the scoped branch changes and push `BIG-GO-194` to `origin`.

## Acceptance Criteria

- Python-named residual scripts/wrappers in the touched ops/helper surface are removed or replaced with non-Python equivalents.
- Canonical documentation and config no longer point at removed Python helper paths in the touched area.
- The changed wrapper/CLI flows still execute through the Go-backed toolchain.
- Validation is recorded with exact commands and concrete pass/fail results.
- The issue branch is committed and pushed.

## Validation

- Search for remaining references to the renamed helper paths.
- Run targeted `go test` coverage for `bigclawctl` command behavior.
- Execute the updated shell helper entrypoints with `--help` or equivalent smoke commands.
- Capture `git status --short` before commit and after validation.

## Validation Results

- `go test ./cmd/bigclawctl` (in `bigclaw-go`) -> `ok  	bigclaw-go/cmd/bigclawctl	0.525s`
- `bash scripts/dev-smoke` -> passed; executed `go test ./...` under `bigclaw-go` with all listed packages passing
- `bash scripts/ops/bigclaw-github-sync status --json` -> exited `0` with `status: "ok"` on branch `BIG-GO-194`
- `bash scripts/ops/bigclaw-refill-queue --local-issues local-issues.json` -> exited `0`; dry-run reported `queue_drained: true` and no candidates
- `bash scripts/ops/bigclaw-workspace-bootstrap --help` -> `usage: bigclawctl workspace <bootstrap|cleanup|validate> [flags]`
- `bash scripts/ops/symphony-workspace-bootstrap --help` -> `usage: bigclawctl workspace <bootstrap|cleanup|validate> [flags]`
- `bash scripts/ops/symphony-workspace-validate --help` -> exited `0` and printed `bigclawctl workspace validate` flag help
- `rg -n 'scripts/ops/(bigclaw-github-sync|bigclaw-refill-queue|bigclaw-workspace-bootstrap|symphony-workspace-bootstrap|symphony-workspace-validate)|scripts/dev-smoke' README.md docs pyproject.toml scripts` -> only the new non-Python helper paths remain in active docs/scripts
- `find scripts bigclaw-go/scripts -type f \( -name '*.py' -o -name '*.pyw' \) | sort` -> only Python test modules remain under the script directories after the second sweep
- `rg -n 'create_issues\.py|run_matrix\.py|soak_local\.py|capacity_certification\.py|broker_failover_stub_matrix\.py|cross_process_coordination_surface\.py|export_validation_bundle\.py|external_store_validation\.py|mixed_workload_matrix\.py|multi_node_shared_queue\.py|run_task_smoke\.py|subscriber_takeover_fault_matrix\.py|validation_bundle_continuation_policy_gate\.py|validation_bundle_continuation_scorecard\.py|shadow_compare\.py|shadow_matrix\.py|live_shadow_scorecard\.py|export_live_shadow_bundle\.py' bigclaw-go scripts README.md docs workflow.md` -> no matches in active docs/scripts after entrypoint normalization
- `python3 -m py_compile scripts/create_issues bigclaw-go/scripts/benchmark/capacity_certification bigclaw-go/scripts/benchmark/run_matrix bigclaw-go/scripts/benchmark/soak_local bigclaw-go/scripts/e2e/broker_failover_stub_matrix bigclaw-go/scripts/e2e/cross_process_coordination_surface bigclaw-go/scripts/e2e/export_validation_bundle bigclaw-go/scripts/e2e/external_store_validation bigclaw-go/scripts/e2e/mixed_workload_matrix bigclaw-go/scripts/e2e/multi_node_shared_queue bigclaw-go/scripts/e2e/run_task_smoke bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard bigclaw-go/scripts/migration/export_live_shadow_bundle bigclaw-go/scripts/migration/live_shadow_scorecard bigclaw-go/scripts/migration/shadow_compare bigclaw-go/scripts/migration/shadow_matrix` -> passed
- `bigclaw-go/scripts/benchmark/run_matrix --help` -> passed
- `bigclaw-go/scripts/e2e/run_task_smoke --help` -> passed
- `bigclaw-go/scripts/migration/shadow_compare --help` -> passed
- `bash -n bigclaw-go/scripts/e2e/run_all.sh bigclaw-go/scripts/e2e/kubernetes_smoke.sh bigclaw-go/scripts/e2e/ray_smoke.sh` -> passed
- `go test ./internal/regression` (in `bigclaw-go`) -> `ok  	bigclaw-go/internal/regression	0.202s`
- `python3 -m unittest bigclaw-go/scripts/benchmark/capacity_certification_test.py bigclaw-go/scripts/e2e/broker_failover_stub_matrix_test.py bigclaw-go/scripts/e2e/export_validation_bundle_test.py bigclaw-go/scripts/e2e/multi_node_shared_queue_test.py bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.py bigclaw-go/scripts/e2e/run_all_test.py` -> failed under system Python 3.9 because `bigclaw-go/scripts/e2e/export_validation_bundle` uses `Path | None` annotations that require Python 3.10+
- Final compatibility sweep:
- `find . -path './.git' -prune -o -type f -name '*.py' -perm -111 -print | sort` -> no executable `.py` files remain
- `bigclaw-go/scripts/e2e/export_validation_bundle --help` -> passed under local Python 3.9 via shell wrapper
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate --help` -> passed under local Python 3.9 via shell wrapper
- `bigclaw-go/scripts/benchmark/run_matrix --help` -> passed under local Python 3.9 via shell wrapper
- `python3 -m py_compile bigclaw-go/scripts/e2e/export_validation_bundle.py bigclaw-go/scripts/e2e/external_store_validation.py` -> passed after adding postponed annotation evaluation
- `python3 -m unittest bigclaw-go/scripts/benchmark/capacity_certification_test.py bigclaw-go/scripts/e2e/broker_failover_stub_matrix_test.py bigclaw-go/scripts/e2e/export_validation_bundle_test.py bigclaw-go/scripts/e2e/multi_node_shared_queue_test.py bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.py bigclaw-go/scripts/e2e/run_all_test.py` -> `Ran 16 tests in 2.864s` / `OK`
- `bigclaw-go/docs/reports/broker-failover-fault-injection-validation-pack.md` follow-on placeholder updated from `scripts/e2e/broker_failover_validation.py` to `scripts/e2e/broker_failover_validation` so the remaining doc surface no longer suggests a Python-named future CLI path
