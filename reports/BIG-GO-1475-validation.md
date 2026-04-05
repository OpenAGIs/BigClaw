# BIG-GO-1475 Validation

## Scope

Collapsed residual Python report-surface helpers in `bigclaw-go/scripts` by removing redundant Python-only tests, porting the active validation-bundle continuation scorecard / policy-gate helpers to Go-owned reporting commands, replacing the shared live-smoke submit/poll helper with a Go-owned entrypoint, moving the live-shadow scorecard / bundle exporters into Go-owned reporting commands, moving the benchmark capacity-certification generator into Go-owned reporting code, porting the live-validation bundle exporter into Go-owned reporting code, porting the shadow compare / matrix migration helpers into Go-owned reporting code, porting the benchmark matrix / local soak helpers into Go-owned reporting code, porting the mixed-workload matrix helper into Go-owned reporting/runtime code, and porting the cross-process coordination capability surface helper into Go-owned reporting code.

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
- `bigclaw-go/scripts/e2e/export_validation_bundle.py`
  - Replaced by Go-owned reporting logic in `bigclaw-go/internal/reporting/live_validation_bundle.go`, coverage in `bigclaw-go/internal/reporting/live_validation_bundle_test.go` plus `bigclaw-go/internal/regression/live_validation_index_test.go` / `bigclaw-go/internal/regression/live_validation_summary_test.go`, and the Go entrypoint `bigclaw-go/scripts/e2e/export_validation_bundle/main.go`.
- `bigclaw-go/scripts/migration/shadow_compare.py`
  - Replaced by Go-owned reporting logic in `bigclaw-go/internal/reporting/shadow_compare_matrix.go`, coverage in `bigclaw-go/internal/reporting/shadow_compare_matrix_test.go`, and the Go entrypoint `bigclaw-go/scripts/migration/shadow_compare/main.go`.
- `bigclaw-go/scripts/migration/shadow_matrix.py`
  - Replaced by Go-owned reporting logic in `bigclaw-go/internal/reporting/shadow_compare_matrix.go`, coverage in `bigclaw-go/internal/reporting/shadow_compare_matrix_test.go`, and the Go entrypoint `bigclaw-go/scripts/migration/shadow_matrix/main.go`.
- `tests/test_shadow_matrix_corpus.py`
  - Replaced by Go-owned corpus-coverage/report coverage in `bigclaw-go/internal/reporting/shadow_compare_matrix_test.go`.
- `bigclaw-go/scripts/benchmark/run_matrix.py`
  - Replaced by Go-owned reporting/runtime logic in `bigclaw-go/internal/reporting/benchmark_matrix.go`, coverage in `bigclaw-go/internal/reporting/benchmark_matrix_test.go`, and the Go entrypoint `bigclaw-go/scripts/benchmark/run_matrix/main.go`.
- `bigclaw-go/scripts/benchmark/soak_local.py`
  - Replaced by Go-owned reporting/runtime logic in `bigclaw-go/internal/reporting/benchmark_matrix.go`, coverage in `bigclaw-go/internal/reporting/benchmark_matrix_test.go`, and the Go entrypoint `bigclaw-go/scripts/benchmark/soak_local/main.go`.
- `bigclaw-go/scripts/e2e/mixed_workload_matrix.py`
  - Replaced by Go-owned reporting/runtime logic in `bigclaw-go/internal/reporting/mixed_workload.go`, coverage in `bigclaw-go/internal/reporting/mixed_workload_test.go`, and the Go entrypoint `bigclaw-go/scripts/e2e/mixed_workload_matrix/main.go`.
- `bigclaw-go/scripts/e2e/cross_process_coordination_surface.py`
  - Replaced by Go-owned reporting logic in `bigclaw-go/internal/reporting/coordination_surface.go`, coverage in `bigclaw-go/internal/reporting/coordination_surface_test.go`, and the Go entrypoint `bigclaw-go/scripts/e2e/cross_process_coordination_surface/main.go`.
- `tests/test_cross_process_coordination_surface.py`
  - Replaced by Go-owned coordination-surface coverage in `bigclaw-go/internal/reporting/coordination_surface_test.go`.

## Delete Conditions

- The remaining Python generator scripts stay in place only where checked-in report artifacts and docs still rely on them as live generation entrypoints.
- The validation-bundle continuation scorecard and policy-gate helpers were deleted because `run_all.sh`, the checked-in report metadata, and the reviewer docs now point at Go-owned entrypoints instead.
- The shared live-smoke submit/poll helper was deleted because the shell wrappers, live-validation docs, and issue-coverage ownership now point at the Go entrypoint instead.
- The live-shadow scorecard and bundle exporters were deleted because the migration docs, checked-in scorecard/bundle artifacts, closeout commands, and regression expectations now point at Go entrypoints instead.
- The benchmark capacity-certification helper was deleted because the benchmark plan, readiness docs, issue-coverage ownership, and checked-in certification matrix now point at the Go entrypoint instead.
- The live-validation bundle exporter was deleted because `run_all.sh`, the README, and the Python report-consumer fixture now invoke the Go entrypoint while the checked-in live-validation summary/index regressions still validate the same artifact surface.
- The shadow compare and shadow matrix helpers were deleted because the migration docs, coverage map, and checked-in live-shadow artifacts now point at Go entrypoints while Go-owned tests validate the compare, matrix, and corpus-coverage behavior directly.
- The benchmark matrix and local soak helpers were deleted because the benchmark plan, readiness reports, issue coverage map, and checked-in capacity-certification consumer now point at Go entrypoints while Go-owned tests validate the benchmark parsing and soak artifact shapes directly.
- The mixed-workload matrix helper was deleted because the e2e validation docs and checked-in mixed-workload report now point at the Go entrypoint while Go-owned tests validate the routing/result artifact shape directly.
- The cross-process coordination surface helper was deleted because the e2e validation docs and follow-up digest now point at the Go entrypoint while Go-owned tests validate the machine-readable coordination surface shape directly.
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
- Python inventory after the sixth deletion slice:
  - `find . -type f -name '*.py' | sort | wc -l` -> `125`
- Python inventory after the seventh deletion slice:
  - `find . -type f -name '*.py' | sort | wc -l` -> `122`
- Python inventory after the eighth deletion slice:
  - `find . -type f -name '*.py' | sort | wc -l` -> `120`
- Python inventory after the ninth deletion slice:
  - `find . -type f -name '*.py' | sort | wc -l` -> `119`
- Python inventory after the tenth deletion slice:
  - `find . -type f -name '*.py' | sort | wc -l` -> `117`
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
- Targeted Go validation for the live-validation bundle exporter slice:
  - `cd bigclaw-go && go test ./internal/reporting ./internal/regression ./scripts/e2e/export_validation_bundle` -> `ok  	bigclaw-go/internal/reporting	1.579s`, `ok  	bigclaw-go/internal/regression	(cached)`, and `?   	bigclaw-go/scripts/e2e/export_validation_bundle	[no test files]`
- Targeted Python/report-consumer validation for the live-validation bundle exporter slice:
  - `PYTHONPATH=src python3 -m pytest tests/test_parallel_validation_bundle.py -q` -> `1 passed`
- Targeted Go validation for the shadow compare / matrix migration slice:
  - `cd bigclaw-go && go test ./internal/reporting ./internal/regression ./scripts/migration/shadow_compare ./scripts/migration/shadow_matrix` -> `ok  	bigclaw-go/internal/reporting	1.270s`, `ok  	bigclaw-go/internal/regression	(cached)`, `?   	bigclaw-go/scripts/migration/shadow_compare	[no test files]`, and `?   	bigclaw-go/scripts/migration/shadow_matrix	[no test files]`
- Targeted Python/report-consumer validation for the shadow compare / matrix migration slice:
  - `PYTHONPATH=src python3 -m pytest tests/test_live_shadow_bundle.py tests/test_live_shadow_scorecard.py -q` -> `2 passed`
- Targeted Go validation for the benchmark matrix / local soak migration slice:
  - `cd bigclaw-go && go test ./internal/reporting ./scripts/benchmark/run_matrix ./scripts/benchmark/soak_local ./scripts/benchmark/capacity_certification` -> `ok  	bigclaw-go/internal/reporting	1.916s`, `?   	bigclaw-go/scripts/benchmark/run_matrix	[no test files]`, `?   	bigclaw-go/scripts/benchmark/soak_local	[no test files]`, and `?   	bigclaw-go/scripts/benchmark/capacity_certification	[no test files]`
- Targeted Go validation for the mixed-workload matrix migration slice:
  - `cd bigclaw-go && go test ./internal/reporting ./scripts/e2e/mixed_workload_matrix ./scripts/benchmark/capacity_certification` -> `ok  	bigclaw-go/internal/reporting	5.544s`, `?   	bigclaw-go/scripts/e2e/mixed_workload_matrix	[no test files]`, and `?   	bigclaw-go/scripts/benchmark/capacity_certification	[no test files]`
- Targeted Go validation for the cross-process coordination surface migration slice:
  - `cd bigclaw-go && go test ./internal/reporting ./internal/regression ./scripts/e2e/cross_process_coordination_surface` -> `ok  	bigclaw-go/internal/reporting	2.940s`, `ok  	bigclaw-go/internal/regression	(cached)`, and `?   	bigclaw-go/scripts/e2e/cross_process_coordination_surface	[no test files]`
- Targeted Python/doc-consumer validation for the cross-process coordination surface migration slice:
  - `PYTHONPATH=src python3 -m pytest tests/test_followup_digests.py -q` -> `2 passed`
