# BIG-GO-13 Legacy Test Contract Sweep D

## Scope

`BIG-GO-13` closes the next deferred Python test-contract slice from
`reports/BIG-GO-948-validation.md`: `tests/test_design_system.py`,
`tests/test_dsl.py`, `tests/test_evaluation.py`, and
`tests/test_parallel_validation_bundle.py`.

This batch is the requested second broad pass over the remaining Python tests
and fixtures. The first three entries are direct package-level test migrations;
the validation-bundle entry is fixture-backed and now has a Go-owned command and
checked-in report fixtures.

## Python Baseline

Repository-wide Python file count: `0`.

This checkout therefore lands as a native-replacement sweep rather than a direct
Python-file deletion batch because there are no physical `.py` assets left to
remove in-branch.

## Deferred Legacy Test Replacements

The sweep-D Go/native replacement registry lives in
`bigclaw-go/internal/migration/legacy_test_contract_sweep_d.go`.

- `tests/test_design_system.py`
  - Go replacements:
    - `bigclaw-go/internal/designsystem/designsystem.go`
    - `bigclaw-go/internal/designsystem/designsystem_test.go`
    - `bigclaw-go/internal/api/expansion_test.go`
  - Evidence:
    - `bigclaw-go/internal/product/console_test.go`
    - `reports/OPE-92-validation.md`
- `tests/test_dsl.py`
  - Go replacements:
    - `bigclaw-go/internal/workflow/definition.go`
    - `bigclaw-go/internal/workflow/definition_test.go`
    - `bigclaw-go/internal/api/expansion.go`
  - Evidence:
    - `bigclaw-go/internal/api/expansion_test.go`
    - `bigclaw-go/cmd/bigclawctl/migration_commands.go`
- `tests/test_evaluation.py`
  - Go replacements:
    - `bigclaw-go/internal/evaluation/evaluation.go`
    - `bigclaw-go/internal/evaluation/evaluation_test.go`
  - Evidence:
    - `bigclaw-go/internal/planning/planning.go`
    - `bigclaw-go/internal/planning/planning_test.go`
- `tests/test_parallel_validation_bundle.py`
  - Go replacements:
    - `bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands.go`
    - `bigclaw-go/internal/regression/parallel_validation_matrix_docs_test.go`
    - `bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go`
  - Evidence:
    - `bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json`
    - `bigclaw-go/docs/reports/validation-bundle-continuation-policy-gate.json`
    - `bigclaw-go/docs/reports/shared-queue-companion-summary.json`

## Why This Sweep Is Now Safe

`reports/BIG-GO-948-validation.md` deferred this slice until the replacement
owners existed as Go-native contracts or checked-in fixtures. Those owners now
exist in-repo:

- design-system behavior is covered by the Go `designsystem` package and the
  expansion payload contract tests.
- DSL behavior is covered by the Go workflow-definition parser and render API.
- evaluation behavior is covered by the Go evaluation package together with
  planning surfaces that reference that evidence.
- validation-bundle continuation is covered by the Go automation bundle command
  and the checked-in continuation scorecard, policy gate, and shared-queue
  companion fixtures.

## Validation Commands And Results

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
  Result: no output; repository-wide Python file count remained `0`.
- `find tests bigclaw-go/docs/reports -type f \\( -name '*.py' -o -name 'validation-bundle-continuation-scorecard.json' -o -name 'validation-bundle-continuation-policy-gate.json' -o -name 'shared-queue-companion-summary.json' \\) 2>/dev/null | sort`
  Result: only the retained Go-owned continuation fixtures were listed:
  `bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/shared-queue-companion-summary.json`,
  `bigclaw-go/docs/reports/shared-queue-companion-summary.json`,
  `bigclaw-go/docs/reports/validation-bundle-continuation-policy-gate.json`, and
  `bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json`.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO13LegacyTestContractSweepD(ManifestMatchesDeferredLegacyTests|ReplacementPathsExist|LaneReportCapturesReplacementState)$'`
  Result: `ok  	bigclaw-go/internal/regression	0.188s`
