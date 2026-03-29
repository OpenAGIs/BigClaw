# BIG-GO-978

## Plan

1. Inventory `bigclaw-go/scripts/benchmark/**` Python files and map each one to its Go-native replacement path.
2. Move the real benchmark matrix and capacity certification logic into `bigclawctl automation benchmark ...`.
3. Remove the migrated Python files in this batch and update docs/references that still point at them.
4. Run targeted Go tests and command-level validation, then measure the repo-wide Python file count delta.
5. Commit the scoped change set and push the issue branch.

## Batch Target Files

- `bigclaw-go/scripts/benchmark/run_matrix.py`
- `bigclaw-go/scripts/benchmark/capacity_certification.py`
- `bigclaw-go/scripts/benchmark/capacity_certification_test.py`
- `bigclaw-go/scripts/benchmark/soak_local.py`

## Acceptance

- Define the exact target file list for this benchmark batch.
- Reduce Python files under `bigclaw-go/scripts/benchmark/` by migrating or removing the batch.
- Preserve a documented replacement path for benchmark matrix, soak-local, and capacity certification workflows.
- Report the effect on total repo Python file count.

## Validation

- `go test ./cmd/bigclawctl`
- Run `bigclawctl automation benchmark run-matrix` against stubbed `go test` / `bigclawctl` helpers in tests.
- Run `bigclawctl automation benchmark capacity-certification --pretty` in tests against checked-in reports.
- `rg --files bigclaw-go/scripts/benchmark -g '*.py'`
- `rg --files -g '*.py' | wc -l`
