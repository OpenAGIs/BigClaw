# BIG-GO-1475 Validation

## Scope

Collapsed residual Python report-surface helpers in `bigclaw-go/scripts` by removing redundant Python-only tests, porting the active validation-bundle continuation scorecard / policy-gate helpers to Go-owned reporting commands, replacing the shared live-smoke submit/poll helper with a Go-owned entrypoint, moving the live-shadow scorecard / bundle exporters into Go-owned reporting commands, and moving the benchmark capacity-certification generator into Go-owned reporting code.

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
- `bigclaw-go/scripts/e2e/run_task_smoke.py`
  - Replaced by Go-owned reporting logic in `bigclaw-go/internal/reporting/task_smoke.go`, coverage in `bigclaw-go/internal/reporting/task_smoke_test.go`, and the Go entrypoint `bigclaw-go/scripts/e2e/run_task_smoke/main.go`.
- `bigclaw-go/scripts/migration/live_shadow_scorecard.py`
  - Replaced by Go-owned reporting logic in `bigclaw-go/internal/reporting/live_shadow.go`, coverage in `bigclaw-go/internal/reporting/live_shadow_test.go`, and the Go entrypoint `bigclaw-go/scripts/migration/live_shadow_scorecard/main.go`.
- `bigclaw-go/scripts/migration/export_live_shadow_bundle.py`
  - Replaced by Go-owned reporting logic in `bigclaw-go/internal/reporting/live_shadow.go`, regression coverage in `bigclaw-go/internal/regression/live_shadow_bundle_surface_test.go`, and the Go entrypoint `bigclaw-go/scripts/migration/export_live_shadow_bundle/main.go`.
- `bigclaw-go/scripts/benchmark/capacity_certification.py`
  - Replaced by Go-owned reporting logic in `bigclaw-go/internal/reporting/capacity.go`, coverage in `bigclaw-go/internal/reporting/capacity_test.go`, and the Go entrypoint `bigclaw-go/scripts/benchmark/capacity_certification/main.go`.

## Delete Conditions

- The remaining Python generator scripts stay in place only where checked-in report artifacts and docs still rely on them as live generation entrypoints.
- The validation-bundle continuation scorecard and policy-gate helpers were deleted because `run_all.sh`, the checked-in report metadata, and the reviewer docs now point at Go-owned entrypoints instead.
- The shared live-smoke submit/poll helper was deleted because the shell wrappers, live-validation docs, and issue-coverage ownership now point at the Go entrypoint instead.
- The live-shadow scorecard and bundle exporters were deleted because the migration docs, checked-in scorecard/bundle artifacts, closeout commands, and regression expectations now point at Go entrypoints instead.
- The benchmark capacity-certification helper was deleted because the benchmark plan, readiness docs, issue-coverage ownership, and checked-in certification matrix now point at the Go entrypoint instead.
- The earlier Python-only report-surface test helpers were deleted because Go-owned tests already validate the same report artifacts and reviewer evidence paths.

## Validation

- Python inventory before deletion:
  - `git ls-tree -r --name-only HEAD | rg '\.py$' | wc -l` -> `138`
- Python inventory after the first deletion slice:
  - `git ls-tree -r --name-only bc6ede9 | rg '\.py$' | wc -l` -> `132`
- Python inventory after the second deletion slice:
  - `find . -type f -name '*.py' | sort | wc -l` -> `130`
- Python inventory after the third deletion slice:
  - `find . -type f -name '*.py' | sort | wc -l` -> `129`
- Python inventory after the fourth deletion slice:
  - `find . -type f -name '*.py' | sort | wc -l` -> `127`
- Python inventory after the fifth deletion slice:
  - `find . -type f -name '*.py' | sort | wc -l` -> `126`
- Targeted Go validation:
  - `cd bigclaw-go && go test ./internal/reporting ./internal/regression` -> `ok  	bigclaw-go/internal/reporting	0.801s` and `ok  	bigclaw-go/internal/regression	1.357s`
- Targeted Go validation after the `run_task_smoke` port:
  - `cd bigclaw-go && go test ./internal/reporting ./internal/regression` -> `ok  	bigclaw-go/internal/reporting	0.957s` and `ok  	bigclaw-go/internal/regression	1.412s`
  - `cd bigclaw-go && go test ./scripts/e2e/run_task_smoke` -> `?   	bigclaw-go/scripts/e2e/run_task_smoke	[no test files]`
  - `cd bigclaw-go && go test ./internal/reporting ./internal/regression ./scripts/e2e/run_task_smoke` -> `ok  	bigclaw-go/internal/reporting	1.279s`, `ok  	bigclaw-go/internal/regression	(cached)`, and `?   	bigclaw-go/scripts/e2e/run_task_smoke	[no test files]`
- Targeted Python validation for migrated references:
  - `PYTHONPATH=src python3 -m pytest tests/test_validation_bundle_continuation_scorecard.py tests/test_validation_bundle_continuation_policy_gate.py tests/test_followup_digests.py -q` -> `8 passed`
- Targeted Python/report-consumer validation for the live-validation bundle surface:
  - `PYTHONPATH=src python3 -m pytest tests/test_parallel_validation_bundle.py -q` -> `1 failed`
  - Failure detail: `/Users/openagi/code/bigclaw-workspaces/BIG-GO-1475/bigclaw-go/scripts/e2e/export_validation_bundle.py` uses `Path | None`, but workspace `python3 --version` is `Python 3.9.6`, so the test aborts with `TypeError: unsupported operand type(s) for |: 'type' and 'NoneType'` before reaching the migrated helper surface.
- Targeted Go validation for the live-shadow migration slice:
  - `cd bigclaw-go && go test ./internal/reporting ./internal/regression ./scripts/migration/live_shadow_scorecard ./scripts/migration/export_live_shadow_bundle` -> `ok  	bigclaw-go/internal/reporting	(cached)`, `ok  	bigclaw-go/internal/regression	(cached)`, `?   	bigclaw-go/scripts/migration/live_shadow_scorecard	[no test files]`, and `?   	bigclaw-go/scripts/migration/export_live_shadow_bundle	[no test files]`
- Targeted Python/report-consumer validation for the live-shadow migration slice:
  - `PYTHONPATH=src python3 -m pytest tests/test_live_shadow_bundle.py tests/test_live_shadow_scorecard.py -q` -> `5 passed`
- Targeted Go validation for the capacity-certification migration slice:
  - `cd bigclaw-go && go test ./internal/reporting ./scripts/benchmark/capacity_certification` -> `ok  	bigclaw-go/internal/reporting	0.859s` and `?   	bigclaw-go/scripts/benchmark/capacity_certification	[no test files]`
  - `cd bigclaw-go && go test ./internal/reporting ./internal/regression ./scripts/benchmark/capacity_certification` -> `ok  	bigclaw-go/internal/reporting	(cached)`, `ok  	bigclaw-go/internal/regression	0.485s`, and `?   	bigclaw-go/scripts/benchmark/capacity_certification	[no test files]`
