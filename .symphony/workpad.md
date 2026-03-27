# BIG-GO-902 Workpad

## Plan

1. Inventory existing Python automation scripts and current Go CLI coverage.
2. Identify the smallest issue-scoped migration slice that produces a concrete executable plan plus first implementation changes.
3. Extend `bigclawctl` with missing Go CLI entry points and compatibility-oriented command structure.
4. Document the migration/compatibility plan, validation commands, regression surface, branch/PR guidance, and risks.
5. Run targeted tests for modified Go CLI areas and record exact commands and results.
6. Commit changes and push the issue branch to remote.

## Acceptance

- Produce an executable migration plan for moving script-layer automation to Go CLI subcommands.
- Deliver first-batch implementation and/or adaptation list with concrete command mappings.
- Define exact validation commands and regression surface.
- Provide branch / PR recommendation and note key risks.

## Validation

- Run targeted Go tests covering modified CLI command registration and behavior.
- If docs or mapping files are added, verify paths and command references against repository layout.
- Record exact commands and whether they passed or failed.

## Validation Results

- `cd bigclaw-go && go test ./cmd/bigclawctl/... ./internal/legacyshim/...`
  - Result: passed
- `cd bigclaw-go && go run ./cmd/bigclawctl legacy-python inventory --json`
  - Result: passed
- `cd bigclaw-go && go run ./cmd/bigclawctl legacy-python inventory --json | rg 'multi_node_shared_queue|subscriber_takeover_fault_matrix|export_validation_bundle'`
  - Result: passed
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e export-validation-bundle --help`
  - Result: passed
- `cd bigclaw-go && python3 scripts/e2e/export_validation_bundle.py --help`
  - Result: passed
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e validation-bundle-scorecard --help`
  - Result: passed
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e validation-bundle-policy-gate --help`
  - Result: passed
- `cd bigclaw-go && python3 scripts/e2e/validation_bundle_continuation_scorecard.py --help`
  - Result: passed
- `cd bigclaw-go && python3 scripts/e2e/validation_bundle_continuation_policy_gate.py --help`
  - Result: passed
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark run-matrix --help`
  - Result: passed
- `cd bigclaw-go && python3 scripts/benchmark/run_matrix.py --help`
  - Result: passed
