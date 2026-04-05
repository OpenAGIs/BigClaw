# BIG-GO-1475 Validation

## Scope

Collapsed residual Python report-surface test helpers in `bigclaw-go/scripts` that duplicated coverage already enforced by Go-owned reporting and regression packages.

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

## Delete Conditions

- The Python generator scripts remain in place because checked-in report artifacts and docs still reference them as generation entrypoints.
- Only the Python-only report-surface test helpers are deleted here because Go-owned tests already validate the corresponding report artifacts and reviewer evidence paths.

## Validation

- Python inventory before deletion:
  - `git ls-tree -r --name-only HEAD | rg '\.py$' | wc -l` -> `138`
- Python inventory after deletion:
  - `find . -type f -name '*.py' | sort | wc -l` -> `132`
- Targeted Go validation:
  - `cd bigclaw-go && go test ./internal/reporting ./internal/regression` -> `ok  	bigclaw-go/internal/reporting	5.132s` and `ok  	bigclaw-go/internal/regression	6.829s`
