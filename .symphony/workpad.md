# BIG-GO-1067 Workpad

## Plan
- Confirm the full `BIG-GO-1067` target list of residual benchmark and e2e Python assets is already absent from `bigclaw-go/scripts/benchmark/` and `bigclaw-go/scripts/e2e/`.
- Verify the Go replacement paths and regression guards that now cover the deleted benchmark and e2e entrypoints.
- Record a fresh validation pass for the umbrella sweep, including current `.py` counts, targeted `go test` runs, and Go CLI help commands for the retained benchmark/e2e entry surfaces.
- Add issue-scoped closeout artifacts summarizing the covered asset list, overall Python-file impact, exact validation commands/results, and remaining risk.
- Commit the issue-scoped evidence on a dedicated branch and push it to `origin`.

## Acceptance
- The batch asset list for `BIG-GO-1067` is explicitly enumerated in repo artifacts.
- The listed Python assets are either deleted or already absent, with active Go or shell replacement entrypoints identified.
- Targeted validation commands and exact results are recorded for the current branch state.
- The response quantifies the net Python-file impact for this sweep.

## Validation
- `find bigclaw-go/scripts/benchmark -maxdepth 1 -name '*.py' | wc -l`
- `find bigclaw-go/scripts/e2e -maxdepth 1 -name '*.py' | wc -l`
- `find . -name '*.py' | wc -l`
- `rg -n "capacity_certification\\.py|capacity_certification_test\\.py|run_matrix\\.py|soak_local\\.py|broker_failover_stub_matrix\\.py|broker_failover_stub_matrix_test\\.py|cross_process_coordination_surface\\.py|export_validation_bundle\\.py|export_validation_bundle_test\\.py|external_store_validation\\.py|mixed_workload_matrix\\.py|multi_node_shared_queue\\.py" .`
- `cd bigclaw-go && go test ./cmd/bigclawctl/... ./internal/regression/...`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark soak-local --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark run-matrix --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark capacity-certification --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e broker-failover-stub-matrix --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e mixed-workload-matrix --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e cross-process-coordination-surface --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e export-validation-bundle --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e external-store-validation --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e multi-node-shared-queue --help`
