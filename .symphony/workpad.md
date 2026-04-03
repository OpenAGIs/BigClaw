# BIG-GO-1120

## Plan
- inventory the candidate benchmark, e2e, and migration Python entrypoints against the current worktree and confirm which surfaces are already Go- or shell-only
- tighten the migration workpad around the real lane scope in this branch: residual delete-gate and final cutover sweeps for `bigclaw-go/scripts/benchmark`, `bigclaw-go/scripts/e2e`, and the removed `bigclaw-go/scripts/migration` lane
- update the Go CLI migration doc so it records `BIG-GO-1120` as the final cutover sweep across benchmark, e2e, and migration entrypoints
- expand regression coverage so docs and script directories fail fast if any of the candidate Python helpers are reintroduced or referenced as active entrypoints
- run targeted Go tests plus repository search/count validation, record exact commands and results, then commit and push the scoped change set

## Acceptance
- the lane file list is explicit and scoped to the candidate residual Python entrypoints from this issue:
- `bigclaw-go/scripts/benchmark/capacity_certification.py`
- `bigclaw-go/scripts/benchmark/capacity_certification_test.py`
- `bigclaw-go/scripts/benchmark/run_matrix.py`
- `bigclaw-go/scripts/benchmark/soak_local.py`
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
- repository-visible active entrypoints for these lanes are Go or shell only, with no live Python helper files in `bigclaw-go/scripts`
- repo references stop treating the candidate Python helpers as active assets and instead point at the Go-native automation commands
- the repository Python file count remains lower than the issue candidate list baseline and is verified explicitly in this worktree
- exact validation commands and outcomes are recorded below

## Validation
- `find bigclaw-go -name '*.py' | wc -l`
- `find bigclaw-go/scripts -maxdepth 2 -type f | sort`
- `rg -n "capacity_certification\\.py|capacity_certification_test\\.py|run_matrix\\.py|soak_local\\.py|broker_failover_stub_matrix\\.py|broker_failover_stub_matrix_test\\.py|cross_process_coordination_surface\\.py|export_validation_bundle\\.py|export_validation_bundle_test\\.py|external_store_validation\\.py|mixed_workload_matrix\\.py|multi_node_shared_queue\\.py|multi_node_shared_queue_test\\.py|run_all_test\\.py|run_task_smoke\\.py|subscriber_takeover_fault_matrix\\.py|validation_bundle_continuation_policy_gate\\.py|validation_bundle_continuation_policy_gate_test\\.py|validation_bundle_continuation_scorecard\\.py|export_live_shadow_bundle\\.py" bigclaw-go/docs/go-cli-script-migration.md bigclaw-go/internal/regression bigclaw-go/cmd/bigclawctl bigclaw-go/scripts`
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/regression`
- `git status --short`

## Validation Results
- `find bigclaw-go -name '*.py' | wc -l` -> `0`
- `find bigclaw-go/scripts -maxdepth 2 -type f | sort` -> `bigclaw-go/scripts/benchmark/run_suite.sh`, `bigclaw-go/scripts/e2e/broker_bootstrap_summary.go`, `bigclaw-go/scripts/e2e/kubernetes_smoke.sh`, `bigclaw-go/scripts/e2e/ray_smoke.sh`, `bigclaw-go/scripts/e2e/run_all.sh`
- `rg -n "capacity_certification\\.py|capacity_certification_test\\.py|run_matrix\\.py|soak_local\\.py|broker_failover_stub_matrix\\.py|broker_failover_stub_matrix_test\\.py|cross_process_coordination_surface\\.py|export_validation_bundle\\.py|export_validation_bundle_test\\.py|external_store_validation\\.py|mixed_workload_matrix\\.py|multi_node_shared_queue\\.py|multi_node_shared_queue_test\\.py|run_all_test\\.py|run_task_smoke\\.py|subscriber_takeover_fault_matrix\\.py|validation_bundle_continuation_policy_gate\\.py|validation_bundle_continuation_policy_gate_test\\.py|validation_bundle_continuation_scorecard\\.py|export_live_shadow_bundle\\.py" bigclaw-go/docs/go-cli-script-migration.md bigclaw-go/internal/regression bigclaw-go/cmd/bigclawctl bigclaw-go/scripts` -> matches only in `bigclaw-go/internal/regression/e2e_entrypoint_migration_test.go`, where the deleted Python paths are asserted as disallowed; no active doc, command, or script references remain
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/regression` -> `ok   bigclaw-go/cmd/bigclawctl 5.179s`; `ok   bigclaw-go/internal/regression 3.479s`
- `git status --short` -> modified `.symphony/workpad.md`, `bigclaw-go/cmd/bigclawctl/automation_commands_test.go`, `bigclaw-go/docs/go-cli-script-migration.md`, `bigclaw-go/internal/regression/e2e_entrypoint_migration_test.go`
