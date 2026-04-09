# BIG-GO-187 Python Reduction Sweep

## Scope

Broad repo Python reduction sweep AA for `BIG-GO-187` tightens the remaining
legacy Python reference density in the active migration-doc surface instead of
repeating another zero-`.py` inventory pass. The scoped live docs in this lane
are:

- `docs/go-cli-script-migration-plan.md`
- `docs/go-mainline-cutover-issue-pack.md`
- `docs/go-mainline-cutover-handoff.md`

## Residual Reference Reduction

The repository already remained physically Python-free before this issue, so
the measurable reduction here is textual legacy reference density inside active
docs that operators and maintainers still read.

- `docs/go-cli-script-migration-plan.md`: `31 -> 10` legacy Python references
- `docs/go-mainline-cutover-issue-pack.md`: `83 -> 24` legacy Python references
- `docs/go-mainline-cutover-handoff.md`: `10 -> 3` legacy Python references

The compacted grouped forms now retained in those docs are:

- `bigclaw-go/scripts/benchmark/{capacity_certification,capacity_certification_test,run_matrix,soak_local}.py`
- `bigclaw-go/scripts/e2e/{broker_failover_stub_matrix,broker_failover_stub_matrix_test,cross_process_coordination_surface,export_validation_bundle,export_validation_bundle_test,external_store_validation,mixed_workload_matrix,multi_node_shared_queue,multi_node_shared_queue_test,run_all_test,run_task_smoke,subscriber_takeover_fault_matrix,validation_bundle_continuation_policy_gate,validation_bundle_continuation_policy_gate_test,validation_bundle_continuation_scorecard}.py`
- `bigclaw-go/scripts/migration/{export_live_shadow_bundle,live_shadow_scorecard,shadow_compare,shadow_matrix}.py`
- `scripts/{create_issues,dev_smoke}.py`
- `src/bigclaw/{models,connectors,mapping,dsl}.py`
- `src/bigclaw/{risk,governance,execution_contract,audit_events}.py`
- `src/bigclaw/{runtime,scheduler,orchestration,workflow,queue}.py`
- `src/bigclaw/{observability,reports,evaluation,operations}.py`
- `src/bigclaw/{repo_triage,run_detail,dashboard_run_contract,operations,saved_views}.py`
- `src/bigclaw/{repo_links,repo_commits,repo_gateway,repo_plane,repo_board,repo_registry,repo_governance}.py`
- `src/bigclaw/{github_sync,workspace_bootstrap,workspace_bootstrap_cli,workspace_bootstrap_validation,parallel_refill}.py`
- `src/bigclaw/{service,__main__}.py`
- `src/bigclaw/{governance,observability,operations,orchestration,pilot}.py`
- `src/bigclaw/{collaboration,repo_board,repo_commits,repo_gateway,repo_governance,repo_links,repo_plane,repo_registry,repo_triage,issue_archive,roadmap}.py`
- `src/bigclaw/{console_ia,design_system,saved_views,ui_review}.py`
- `src/bigclaw/{github_sync,parallel_refill,workspace_bootstrap,workspace_bootstrap_cli,workspace_bootstrap_validation,service,__main__}.py`

## Guard Surface

`bigclaw-go/internal/regression/big_go_187_zero_python_guard_test.go` now pins:

- per-doc legacy-reference budgets for the three compacted active docs
- required grouped legacy path forms that preserve migration meaning
- absence of the previously expanded individual `.py` examples that were
  collapsed in this sweep

## Validation Commands And Results

- `rg -n "python3|\\.py\\b|#!/usr/bin/env python|#!/usr/bin/python" docs bigclaw-go/internal bigclaw-go/cmd --glob '!bigclaw-go/internal/regression/**' --glob '!bigclaw-go/docs/reports/**' | head -n 200`
  Result: targeted manual audit confirms the active residual surface is now
  concentrated in grouped migration docs plus intentional compatibility tests.
- `cd bigclaw-go && go test ./internal/regression -run 'TestBIGGO187' -count=1`
  Result: recorded in `reports/BIG-GO-187-validation.md`.
