# BIG-GO-1053 Workpad

## Plan

1. Inspect `bigclaw-go/scripts/e2e/`, migration docs, README/workflow/hooks/CI references, and regression coverage to confirm the current Go-only state and find stale Python-helper references.
2. Update the scoped migration documentation and regression tests so tranche-2 e2e entrypoints are described only through active Go/shell surfaces, without relying on historical `.py` filenames as the primary contract.
3. Run targeted validation for the regression package and `bigclawctl automation` help surfaces, then record exact commands and results for closeout.
4. Commit the scoped patch and push it to the remote branch for `BIG-GO-1053`.

## Acceptance

- `bigclaw-go/scripts/e2e/` remains Python-free and the repository’s active e2e operator surface is documented as Go/shell only.
- README and migration docs point at current Go/shell entrypoints for the e2e tranche covered by this issue.
- Regression coverage fails if Python helpers reappear in `scripts/e2e/` or if the migration doc stops listing the active Go/shell entrypoints for this tranche.
- Targeted validation commands complete successfully and are recorded with exact results.

## Validation

- `cd bigclaw-go && go test ./internal/regression -run TestE2E`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e run-task-smoke --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e export-validation-bundle --help`

## Validation Results

- `cd bigclaw-go && go test ./internal/regression -run TestE2E` -> `ok  	bigclaw-go/internal/regression	0.457s`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation --help` -> exit `0`; printed `usage: bigclawctl automation <e2e|benchmark|migration> ...`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e run-task-smoke --help` -> exit `0`; printed `usage: bigclawctl automation e2e run-task-smoke [flags]`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e export-validation-bundle --help` -> exit `0`; printed `usage: bigclawctl automation e2e export-validation-bundle [flags]`
- `find bigclaw-go/scripts/e2e -maxdepth 1 -type f -name '*.py' | wc -l` -> `0`
- `find . -type f -name '*.py' | wc -l` -> `17`
- `dirs=(); for p in README.md bigclaw-go/README.md bigclaw-go/docs/go-cli-script-migration.md workflow.md .github/workflows/ci.yml .githooks/post-commit .githooks/post-rewrite; do [ -e "$p" ] && dirs+=("$p"); done; rg -n "scripts/e2e/.*\.py|run_task_smoke\.py|export_validation_bundle\.py|validation_bundle_continuation_scorecard\.py|validation_bundle_continuation_policy_gate\.py|broker_failover_stub_matrix\.py|mixed_workload_matrix\.py|cross_process_coordination_surface\.py|subscriber_takeover_fault_matrix\.py|external_store_validation\.py|multi_node_shared_queue\.py" "${dirs[@]}"` -> exit `1` with no matches

## Notes

- `bigclaw-go/scripts/e2e/` was already Python-free when this branch started from `main`, so this issue’s scoped change set hardens docs/CI/regression coverage around the Go-only tranche-2 entrypoints rather than deleting additional e2e `.py` files in-place.
