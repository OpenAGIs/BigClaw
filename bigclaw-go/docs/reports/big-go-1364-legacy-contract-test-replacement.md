# BIG-GO-1364 Legacy Contract Test Replacement Sweep A

`BIG-GO-1364` closes out a contract-oriented slice of removed Python tests by
recording the checked-in Go/native owners that now carry those assertions.

## Baseline

- Repository-wide Python file count: `0`.
- `find . -name '*.py' | wc -l` cannot go lower on this branch, so acceptance
  for this lane is concrete Go/native replacement evidence in git.

## Go/Native Replacement Artifact

- Registry path: `bigclaw-go/internal/migration/legacy_contract_tests.go`
- Purpose: track the removed Python contract tests in sweep A and the active
  Go/native surfaces that replaced them.

## Replacement Mapping

### `tests/test_execution_contract.py`

- Replacement kind: `go-contract-owner`
- Active native owners:
  - `bigclaw-go/internal/contract/execution.go`
  - `bigclaw-go/internal/contract/execution_test.go`
- Checked-in evidence:
  - `docs/go-mainline-cutover-issue-pack.md`
  - `reports/OPE-131-validation.md`

### `tests/test_dashboard_run_contract.py`

- Replacement kind: `go-contract-owner`
- Active native owners:
  - `bigclaw-go/internal/product/dashboard_run_contract.go`
  - `bigclaw-go/internal/product/dashboard_run_contract_test.go`
- Checked-in evidence:
  - `bigclaw-go/internal/api/expansion_test.go`
  - `reports/OPE-129-validation.md`

### `tests/test_cross_process_coordination_surface.py`

- Replacement kind: `go-api-surface`
- Active native owners:
  - `bigclaw-go/internal/api/coordination_surface.go`
  - `bigclaw-go/internal/regression/coordination_contract_surface_test.go`
- Checked-in evidence:
  - `bigclaw-go/docs/e2e-validation.md`
  - `bigclaw-go/docs/reports/cross-process-coordination-capability-surface.json`

### `tests/test_followup_digests.py`

- Replacement kind: `repo-native-report-guard`
- Active native owners:
  - `bigclaw-go/docs/reports/parallel-follow-up-index.md`
  - `bigclaw-go/internal/regression/followup_index_docs_test.go`
- Checked-in evidence:
  - `docs/parallel-refill-queue.md`
  - `reports/OPE-270-271-validation.md`

### `tests/test_parallel_refill.py`

- Replacement kind: `go-queue-and-native-docs`
- Active native owners:
  - `bigclaw-go/internal/refill/queue_test.go`
  - `bigclaw-go/internal/regression/parallel_validation_matrix_docs_test.go`
- Checked-in evidence:
  - `docs/parallel-refill-queue.json`
  - `reports/OPE-270-271-validation.md`

## Regression Guard

- `bigclaw-go/internal/regression/big_go_1364_legacy_contract_test_replacement_test.go`
  verifies the replacement registry contents, confirms the retired Python tests
  stay absent, validates the referenced native paths, and checks this lane
  report.

## Validation Commands

- `find . -name '*.py' | wc -l`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1364LegacyContractTestReplacement(ManifestMatchesRetiredTests|PathsExist|LaneReportCapturesReplacementState)$'`
