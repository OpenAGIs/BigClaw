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
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark capacity-certification --help`
  - Result: passed
- `cd bigclaw-go && python3 scripts/benchmark/capacity_certification.py --help`
  - Result: passed
- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration live-shadow-scorecard --help`
  - Result: passed
- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration export-live-shadow-bundle --help`
  - Result: passed
- `cd bigclaw-go && python3 scripts/migration/live_shadow_scorecard.py --help`
  - Result: passed
- `cd bigclaw-go && python3 scripts/migration/export_live_shadow_bundle.py --help`
  - Result: passed
- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration shadow-matrix --help`
  - Result: passed
- `cd bigclaw-go && python3 scripts/migration/shadow_matrix.py --help`
  - Result: passed
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e cross-process-coordination-surface --help`
  - Result: passed
- `cd bigclaw-go && python3 scripts/e2e/cross_process_coordination_surface.py --help`
  - Result: passed
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e mixed-workload-matrix --help`
  - Result: passed
- `cd bigclaw-go && python3 scripts/e2e/mixed_workload_matrix.py --help`
  - Result: passed
- `cd bigclaw-go && go run ./cmd/bigclawctl legacy-python inventory --json | jq -r '.entries[] | select(.script_path=="bigclaw-go/scripts/e2e/mixed_workload_matrix.py") | [.script_path,.status,.compatibility_layer] | @tsv'`
  - Result: passed
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e external-store-validation --help`
  - Result: passed
- `cd bigclaw-go && python3 scripts/e2e/external_store_validation.py --help`
  - Result: passed
- `cd bigclaw-go && go run ./cmd/bigclawctl legacy-python inventory --json | jq -r '.entries[] | select(.script_path=="bigclaw-go/scripts/e2e/external_store_validation.py") | [.script_path,.status,.compatibility_layer] | @tsv'`
  - Result: passed
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e subscriber-takeover-fault-matrix --help`
  - Result: passed
- `cd bigclaw-go && python3 scripts/e2e/subscriber_takeover_fault_matrix.py --help`
  - Result: passed
- `cd bigclaw-go && go run ./cmd/bigclawctl legacy-python inventory --json | jq -r '.entries[] | select(.script_path=="bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py") | [.script_path,.status,.compatibility_layer] | @tsv'`
  - Result: passed
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e broker-failover-stub-matrix --help`
  - Result: passed
- `cd bigclaw-go && python3 scripts/e2e/broker_failover_stub_matrix.py --help`
  - Result: passed
- `cd bigclaw-go && go run ./cmd/bigclawctl legacy-python inventory --json | jq -r '.entries[] | select(.script_path=="bigclaw-go/scripts/e2e/broker_failover_stub_matrix.py") | [.script_path,.status,.compatibility_layer] | @tsv'`
  - Result: passed
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e multi-node-shared-queue --help`
  - Result: passed
- `cd bigclaw-go && python3 scripts/e2e/multi_node_shared_queue.py --help`
  - Result: passed
- `cd bigclaw-go && tmpdir=$(mktemp -d) && go run ./cmd/bigclawctl automation e2e subscriber-takeover-fault-matrix --go-root "$tmpdir" --output docs/reports/takeover.json && test -f "$tmpdir/docs/reports/takeover.json"`
  - Result: passed
- `cd bigclaw-go && tmpdir=$(mktemp -d) && go run ./cmd/bigclawctl automation e2e broker-failover-stub-matrix --repo-root "$tmpdir" --output docs/reports/broker.json --artifact-root docs/reports/broker-artifacts --checkpoint-fencing-summary-output docs/reports/checkpoint.json --retention-boundary-summary-output docs/reports/retention.json && test -f "$tmpdir/docs/reports/broker.json" && test -f "$tmpdir/docs/reports/checkpoint.json" && test -f "$tmpdir/docs/reports/retention.json"`
  - Result: passed
- `cd bigclaw-go && tmpdir=$(mktemp -d) && mkdir -p "$tmpdir/docs/reports" && printf '%s\n' '{"count":2,"cross_node_completions":1,"duplicate_completed_tasks":[],"duplicate_started_tasks":[]}' > "$tmpdir/docs/reports/multi-node-shared-queue-report.json" && printf '%s\n' '{"summary":{"scenario_count":3,"passing_scenarios":3,"duplicate_delivery_count":1,"stale_write_rejections":1}}' > "$tmpdir/docs/reports/multi-subscriber-takeover-validation-report.json" && printf '%s\n' '{"summary":{"scenario_count":2,"passing_scenarios":2,"stale_write_rejections":1}}' > "$tmpdir/docs/reports/live-multi-node-subscriber-takeover-report.json" && go run ./cmd/bigclawctl automation e2e cross-process-coordination-surface --repo-root "$tmpdir" --multi-node-report docs/reports/multi-node-shared-queue-report.json --takeover-report docs/reports/multi-subscriber-takeover-validation-report.json --live-takeover-report docs/reports/live-multi-node-subscriber-takeover-report.json --output docs/reports/cross.json && test -f "$tmpdir/docs/reports/cross.json"`
  - Result: passed
- `cd bigclaw-go && tmpdir=$(mktemp -d) && go run ./cmd/bigclawctl automation e2e multi-node-shared-queue --go-root . --report-path "$tmpdir/shared.json" --takeover-report-path "$tmpdir/takeover.json" --takeover-artifact-dir "$tmpdir/artifacts" --count 6 --submit-workers 2 --timeout-seconds 30 --takeover-ttl 1s && test -f "$tmpdir/shared.json" && test -f "$tmpdir/takeover.json"`
  - Result: passed
