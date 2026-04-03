# BIG-GO-1102

## Plan
- confirm the current `bigclaw-go/scripts` automation surface is already Python-free and locate live residual references to the removed script tranche
- update the migration/regression guardrails so they assert the current Go/shell-only script surface without carrying stale Python file manifests longer than needed
- remove a small self-contained cluster of legacy Python source assets that already have Go-native owners and are only retained as planning evidence
- update planning/docs references so they point at the Go-native ownership/tests instead of deleted Python source files
- run targeted validation for the affected Go planning/regression surfaces plus repository Python-file counts
- commit the scoped change set and push the branch

## Acceptance
- lane coverage is explicit: removed `bigclaw-go/scripts` Python automation files stay absent and live references are cleaned up
- the change deletes real Python source assets rather than only editing tracker/docs cosmetics
- `find . -name '*.py' | wc -l` decreases from the pre-change baseline of `17`
- exact validation commands and results are recorded below

## Validation
- `find . -name '*.py' | wc -l`
- `rg -n "bigclaw-go/scripts/.+\\.py|scripts/e2e/.+\\.py|scripts/benchmark/.+\\.py|scripts/migration/.+\\.py|run_task_smoke\\.py|export_validation_bundle\\.py|validation_bundle_continuation_scorecard\\.py|validation_bundle_continuation_policy_gate\\.py|broker_failover_stub_matrix\\.py|mixed_workload_matrix\\.py|cross_process_coordination_surface\\.py|subscriber_takeover_fault_matrix\\.py|external_store_validation\\.py|multi_node_shared_queue\\.py|capacity_certification\\.py|run_matrix\\.py|soak_local\\.py|shadow_compare\\.py|shadow_matrix\\.py|live_shadow_scorecard\\.py|export_live_shadow_bundle\\.py" bigclaw-go README.md docs workflow.md -g '!**/*.json' -g '!docs/go-cli-script-migration-plan.md' -g '!bigclaw-go/internal/regression/e2e_entrypoint_migration_test.go'`
- `cd bigclaw-go && go test ./internal/regression ./internal/planning`
- `git status --short`

## Validation Results
- pre-change `find . -name '*.py' | wc -l` -> `17`
- post-change `find . -name '*.py' | wc -l` -> `12`
- `rg -n "bigclaw-go/scripts/.+\\.py|scripts/e2e/.+\\.py|scripts/benchmark/.+\\.py|scripts/migration/.+\\.py|run_task_smoke\\.py|export_validation_bundle\\.py|validation_bundle_continuation_scorecard\\.py|validation_bundle_continuation_policy_gate\\.py|broker_failover_stub_matrix\\.py|mixed_workload_matrix\\.py|cross_process_coordination_surface\\.py|subscriber_takeover_fault_matrix\\.py|external_store_validation\\.py|multi_node_shared_queue\\.py|capacity_certification\\.py|run_matrix\\.py|soak_local\\.py|shadow_compare\\.py|shadow_matrix\\.py|live_shadow_scorecard\\.py|export_live_shadow_bundle\\.py" bigclaw-go README.md docs workflow.md -g '!**/*.json' -g '!docs/go-cli-script-migration-plan.md' -g '!bigclaw-go/internal/regression/e2e_entrypoint_migration_test.go'` -> one intentional documentation match in `bigclaw-go/docs/go-cli-script-migration.md` noting that the removed candidate `.py` entrypoints stay absent; no live README/workflow/runtime references remained
- `cd bigclaw-go && go test ./internal/regression ./internal/planning` -> `ok   bigclaw-go/internal/regression 3.202s`; `ok   bigclaw-go/internal/planning 3.270s`
- `git status --short` -> scoped issue edits are present, and unrelated concurrent worktree changes also exist outside this issue slice (`README.md`, `src/bigclaw/runtime.py`, `src/bigclaw/deprecation.py`, `bigclaw-go/docs/go-cli-script-migration.md`, `bigclaw-go/internal/regression/runtime_residue_purge_test.go`)
