# BIG-GO-112 Residual Test Contract Sweep M

## Scope

`BIG-GO-112` closes the remaining deferred Python test-contract cluster from
`reports/BIG-GO-948-validation.md` that still lacked dedicated replacement
registry artifacts after the broad deletion sweeps:

- `tests/test_design_system.py`
- `tests/test_dsl.py`
- `tests/test_evaluation.py`

These files are already physically absent in the repository. This lane records
the surviving Go-native owners so future residual sweeps can verify the deleted
Python tests were replaced by maintained contract surfaces rather than by
absence-only assertions.

## Python Baseline

Repository-wide Python file count: `0`.

This sweep therefore lands as a replacement-registry and regression-evidence
slice rather than a direct file-deletion batch.

## Residual Contract Replacements

The replacement registry lives in
`bigclaw-go/internal/migration/residual_test_contract_sweep_m.go`.

- `tests/test_design_system.py`
  - Go replacements:
    - `bigclaw-go/internal/designsystem/designsystem.go`
    - `bigclaw-go/internal/product/console.go`
  - Evidence:
    - `bigclaw-go/internal/designsystem/designsystem_test.go`
    - `bigclaw-go/internal/product/console_test.go`
    - `docs/issue-plan.md`
- `tests/test_dsl.py`
  - Go replacements:
    - `bigclaw-go/internal/workflow/definition.go`
    - `bigclaw-go/internal/workflow/closeout.go`
  - Evidence:
    - `bigclaw-go/internal/workflow/definition_test.go`
    - `docs/go-domain-intake-parity-matrix.md`
    - `docs/issue-plan.md`
- `tests/test_evaluation.py`
  - Go replacements:
    - `bigclaw-go/internal/evaluation/evaluation.go`
    - `bigclaw-go/internal/planning/planning.go`
  - Evidence:
    - `bigclaw-go/internal/evaluation/evaluation_test.go`
    - `bigclaw-go/internal/planning/planning_test.go`
    - `docs/go-mainline-cutover-issue-pack.md`

## Why This Sweep Is Safe

- the design-system contract is now represented by a Go-native component
  library, audit model, and console design-system surface.
- the former Python DSL contract is now represented by the Go workflow
  definition parser and closeout orchestration model.
- the former Python evaluation contract is now represented by the Go evaluation
  runner, replay reporting, and planning-owned benchmark evidence surfaces.

`reports/BIG-GO-948-validation.md` deferred these tests until the owning Go
surfaces were mature enough to cite directly. Those owners now exist in-repo
and already carry targeted Go coverage.

## Validation Commands And Results

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
  Result: no output; repository-wide Python file count remained `0`.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO112ResidualTestContractSweepM(ManifestMatchesRetiredTests|ReplacementPathsExist|LaneReportCapturesReplacementState)$'`
  Result: `ok  	bigclaw-go/internal/regression	4.466s`
