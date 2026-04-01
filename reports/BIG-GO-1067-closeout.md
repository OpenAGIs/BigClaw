# BIG-GO-1067 Closeout Index

Issue: `BIG-GO-1067`

Title: `bigclaw-go scripts e2e residual sweep A`

Date: `2026-04-01`

## Branch

`symphony/BIG-GO-1067-validation`

## Latest Code Migration Commit

`d36a1c70`

## In-Repo Artifacts

- Validation report:
  - `reports/BIG-GO-1067-validation.md`
- Machine-readable status:
  - `reports/BIG-GO-1067-status.json`
- Workpad:
  - `.symphony/workpad.md`
- Prior tranche evidence:
  - `reports/BIG-GO-1051-validation.md`
  - `reports/BIG-GO-1053-validation.md`
- Live regression guards:
  - `bigclaw-go/cmd/bigclawctl/automation_commands_test.go`
  - `bigclaw-go/internal/regression/e2e_entrypoint_migration_test.go`

## Outcome

- The full `BIG-GO-1067` target list of 12 residual benchmark and e2e Python assets is
  absent from the current repo surface.
- Benchmark replacements are the Go-native `bigclawctl automation benchmark ...` commands
  plus `bigclaw-go/scripts/benchmark/run_suite.sh`.
- E2E replacements are the Go-native `bigclawctl automation e2e ...` commands plus the
  retained shell wrappers under `bigclaw-go/scripts/e2e/`.
- This branch adds the missing issue-scoped validation and closeout trail for the umbrella
  sweep; the underlying code migration was already landed via `BIG-GO-1051` and
  `BIG-GO-1053`.

## Validation Commands

```bash
find bigclaw-go/scripts/benchmark -maxdepth 1 -name '*.py' | wc -l
find bigclaw-go/scripts/e2e -maxdepth 1 -name '*.py' | wc -l
find . -name '*.py' | wc -l
rg -n "capacity_certification\.py|capacity_certification_test\.py|run_matrix\.py|soak_local\.py|broker_failover_stub_matrix\.py|broker_failover_stub_matrix_test\.py|cross_process_coordination_surface\.py|export_validation_bundle\.py|export_validation_bundle_test\.py|external_store_validation\.py|mixed_workload_matrix\.py|multi_node_shared_queue\.py" tests bigclaw-go docs README.md workflow.md .github . -g '!reports/**' -g '!.symphony/workpad.md' 2>/dev/null
cd bigclaw-go && go test ./cmd/bigclawctl/... ./internal/regression/...
cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark soak-local --help
cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark run-matrix --help
cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark capacity-certification --help
cd bigclaw-go && ./scripts/benchmark/run_suite.sh
cd bigclaw-go && go run ./cmd/bigclawctl automation e2e broker-failover-stub-matrix --help
cd bigclaw-go && go run ./cmd/bigclawctl automation e2e mixed-workload-matrix --help
cd bigclaw-go && go run ./cmd/bigclawctl automation e2e cross-process-coordination-surface --help
cd bigclaw-go && go run ./cmd/bigclawctl automation e2e export-validation-bundle --help
cd bigclaw-go && go run ./cmd/bigclawctl automation e2e external-store-validation --help
cd bigclaw-go && go run ./cmd/bigclawctl automation e2e multi-node-shared-queue --help
```

## Remaining Risk

No blocking repo action remains for `BIG-GO-1067`.

The only caveat is historical: this umbrella lane validates and records a batch that was
already materially migrated on `main`, so the branch contributes issue-scoped evidence and
not a fresh source-file deletion. Archived tracker/report references to the retired Python
filenames remain by design.
