# BIG-GO-1475 Validation

## Scope

Collapsed residual Python report-surface helpers in `bigclaw-go/scripts` by removing redundant Python-only tests and porting the active validation-bundle continuation scorecard / policy-gate helpers to Go-owned reporting commands.

## Deleted Python Files And Replacements

- `bigclaw-go/scripts/benchmark/capacity_certification_test.py`
  - Replaced by Go-owned report coverage in `bigclaw-go/internal/reporting/reporting_test.go` and repo-native artifact checks in `bigclaw-go/docs/reports/capacity-certification-matrix.json`.
- `bigclaw-go/scripts/e2e/broker_failover_stub_matrix_test.py`
  - Replaced by Go-owned broker/report artifact coverage in `bigclaw-go/internal/regression/broker_validation_summary_test.go` and `bigclaw-go/internal/regression/durability_review_bundle_test.go`.
- `bigclaw-go/scripts/e2e/export_validation_bundle_test.py`
  - Replaced by Go-owned live-validation artifact coverage in `bigclaw-go/internal/regression/live_validation_index_test.go`, `bigclaw-go/internal/regression/live_validation_summary_test.go`, and `bigclaw-go/internal/regression/runtime_report_followup_docs_test.go`.
- `bigclaw-go/scripts/e2e/multi_node_shared_queue_test.py`
  - Replaced by Go-owned shared-queue report coverage in `bigclaw-go/internal/regression/shared_queue_report_test.go` and `bigclaw-go/internal/regression/shared_queue_companion_summary_test.go`.
- `bigclaw-go/scripts/e2e/run_all_test.py`
  - Replaced by Go-owned live-validation/run-all report expectations in `bigclaw-go/internal/regression/live_validation_index_test.go` and `bigclaw-go/internal/regression/live_validation_summary_test.go`.
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.py`
  - Replaced by Go-owned continuation-gate/report artifact coverage in `bigclaw-go/internal/regression/runtime_report_followup_docs_test.go` and `bigclaw-go/internal/regression/bundle_followup_index_docs_test.go`.
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py`
  - Replaced by Go-owned reporting logic in `bigclaw-go/internal/reporting/continuation.go` and the Go entrypoint `bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard/main.go`.
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py`
  - Replaced by Go-owned reporting logic in `bigclaw-go/internal/reporting/continuation.go` and the Go entrypoint `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate/main.go`.

## Delete Conditions

- The remaining Python generator scripts stay in place only where checked-in report artifacts and docs still rely on them as live generation entrypoints.
- The validation-bundle continuation scorecard and policy-gate helpers were deleted because `run_all.sh`, the checked-in report metadata, and the reviewer docs now point at Go-owned entrypoints instead.
- The earlier Python-only report-surface test helpers were deleted because Go-owned tests already validate the same report artifacts and reviewer evidence paths.

## Validation

- Python inventory before deletion:
  - `git ls-tree -r --name-only HEAD | rg '\.py$' | wc -l` -> `138`
- Python inventory after the first deletion slice:
  - `git ls-tree -r --name-only bc6ede9 | rg '\.py$' | wc -l` -> `132`
- Python inventory after the second deletion slice:
  - `find . -type f -name '*.py' | sort | wc -l` -> `130`
- Targeted Go validation:
  - `cd bigclaw-go && go test ./internal/reporting ./internal/regression` -> `ok  	bigclaw-go/internal/reporting	0.801s` and `ok  	bigclaw-go/internal/regression	1.357s`
- Targeted Python validation for migrated references:
  - `PYTHONPATH=src python3 -m pytest tests/test_validation_bundle_continuation_scorecard.py tests/test_validation_bundle_continuation_policy_gate.py tests/test_followup_digests.py -q` -> `8 passed`
