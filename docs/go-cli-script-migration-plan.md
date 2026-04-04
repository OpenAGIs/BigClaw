# Go CLI Script Migration Plan

## BIG-GO-1165 sweep

This sweep retires a large residual batch of checked-in Python entrypoints that
already have Go-native runtime coverage or checked-in repo-native artifact
surfaces.

## Replacement paths

- `scripts/dev_smoke.py` -> `cd bigclaw-go && go test ./...` and `cd bigclaw-go && go run ./cmd/bigclawd`
- `scripts/create_issues.py` -> `cd bigclaw-go && go run ./cmd/bigclawctl local-issues --help` and `cd bigclaw-go && go run ./cmd/bigclawctl refill seed --help`
- `bigclaw-go/scripts/benchmark/run_matrix.py` -> `cd bigclaw-go && go test -bench . ./internal/queue ./internal/scheduler`
- `bigclaw-go/scripts/benchmark/soak_local.py` -> checked-in soak artifacts under `bigclaw-go/docs/reports/soak-local-*.json` plus `bigclaw-go/docs/reports/long-duration-soak-report.md`
- `bigclaw-go/scripts/benchmark/capacity_certification.py` -> checked-in admission artifacts under `bigclaw-go/docs/reports/capacity-certification-matrix.json` and `bigclaw-go/docs/reports/capacity-certification-report.md`
- `bigclaw-go/scripts/e2e/broker_failover_stub_matrix.py` -> `cd bigclaw-go && go test ./internal/regression -run TestBrokerValidationSummaryStaysAligned -count=1`
- `bigclaw-go/scripts/e2e/run_task_smoke.py` -> `cd bigclaw-go && go run ./cmd/bigclawctl e2e run-task-smoke --help`
- `bigclaw-go/scripts/e2e/export_validation_bundle.py` -> `cd bigclaw-go && go run ./cmd/bigclawctl e2e export-validation-bundle --help`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py` -> `cd bigclaw-go && go run ./cmd/bigclawctl e2e validation-bundle-continuation-scorecard --pretty`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py` -> `cd bigclaw-go && go run ./cmd/bigclawctl e2e validation-bundle-continuation-policy-gate --pretty`
- `bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py` -> `cd bigclaw-go && go run ./cmd/bigclawctl e2e subscriber-takeover-fault-matrix --pretty`
- `bigclaw-go/scripts/e2e/mixed_workload_matrix.py` -> checked-in `bigclaw-go/docs/reports/mixed-workload-matrix-report.json` plus admission-policy coverage in `bigclaw-go/internal/api/server_test.go`
- `bigclaw-go/scripts/e2e/external_store_validation.py` -> `cd bigclaw-go && go test ./internal/regression -run TestExternalStoreValidationReportStaysAligned -count=1`
- `bigclaw-go/scripts/e2e/cross_process_coordination_surface.py` -> `cd bigclaw-go && go test ./internal/regression -run TestCrossProcessCoordinationReadinessDocsStayAligned -count=1`
- `bigclaw-go/scripts/migration/shadow_compare.py` -> checked-in `bigclaw-go/docs/reports/shadow-compare-report.json` plus `cd bigclaw-go && go test ./internal/regression -run TestLiveShadowScorecardBundleStaysAligned -count=1`
- `bigclaw-go/scripts/migration/shadow_matrix.py` -> checked-in `bigclaw-go/docs/reports/shadow-matrix-report.json` plus `cd bigclaw-go && go test ./internal/regression -run TestLiveShadowScorecardBundleStaysAligned -count=1`
- `bigclaw-go/scripts/migration/live_shadow_scorecard.py` -> checked-in `bigclaw-go/docs/reports/live-shadow-mirror-scorecard.json` plus `cd bigclaw-go && go test ./internal/regression -run TestLiveShadowScorecardBundleStaysAligned -count=1`
- `bigclaw-go/scripts/migration/export_live_shadow_bundle.py` -> checked-in `bigclaw-go/docs/reports/live-shadow-summary.json` and `bigclaw-go/docs/reports/live-shadow-index.json` plus `cd bigclaw-go && go test ./internal/regression -run TestLiveShadowBundleSummaryAndIndexStayAligned -count=1`

## Retired Python test/support entrypoints

- `bigclaw-go/scripts/benchmark/capacity_certification_test.py`
- `bigclaw-go/scripts/e2e/broker_failover_stub_matrix_test.py`
- `bigclaw-go/scripts/e2e/export_validation_bundle_test.py`
- `bigclaw-go/scripts/e2e/multi_node_shared_queue_test.py`
- `bigclaw-go/scripts/e2e/run_all_test.py`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.py`
- `tests/test_parallel_validation_bundle.py`
- `tests/test_validation_bundle_continuation_policy_gate.py`
- `tests/test_validation_bundle_continuation_scorecard.py`
- `tests/test_subscriber_takeover_harness.py`

These were Python-only harnesses for now-repo-native checked-in artifacts and do
not remain on the active Go mainline.

Additional retired Python-only coverage harnesses in `tests/` are removed when
they only exercised these frozen generator scripts instead of an active Go
surface.

## Validation notes

- The repo-level Python count must decrease from the pre-sweep baseline of `138`.
- The active Go replacement path for this sweep is validated with targeted `go test`
  runs and the checked-in report surfaces above.
