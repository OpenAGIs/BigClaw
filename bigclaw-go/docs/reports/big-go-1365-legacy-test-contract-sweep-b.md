# BIG-GO-1365 Legacy Test Contract Sweep B

## Scope

`BIG-GO-1365` closes the deferred legacy Python test-contract slice called out in `reports/BIG-GO-948-validation.md`: `tests/test_control_center.py`, `tests/test_operations.py`, and `tests/test_ui_review.py`.

## Python Baseline

Repository-wide Python file count: `0`.

This checkout therefore lands as a native-replacement sweep rather than a direct Python-file deletion batch because there are no physical `.py` assets left to remove in-branch.

## Deferred Legacy Test Replacements

The sweep-B Go/native replacement registry lives in `bigclaw-go/internal/migration/legacy_test_contract_sweep_b.go`.

- `tests/test_control_center.py`
  - Go replacements:
    - `bigclaw-go/internal/control/controller.go`
    - `bigclaw-go/internal/api/server.go`
    - `bigclaw-go/internal/api/v2.go`
  - Evidence:
    - `bigclaw-go/internal/control/controller_test.go`
    - `bigclaw-go/docs/reports/v2-phase1-operations-foundation-report.md`
- `tests/test_operations.py`
  - Go replacements:
    - `bigclaw-go/internal/product/dashboard_run_contract.go`
    - `bigclaw-go/internal/contract/execution.go`
    - `bigclaw-go/internal/control/controller.go`
  - Evidence:
    - `bigclaw-go/internal/product/dashboard_run_contract_test.go`
    - `bigclaw-go/internal/contract/execution_test.go`
    - `bigclaw-go/docs/reports/v2-phase1-operations-foundation-report.md`
- `tests/test_ui_review.py`
  - Go replacements:
    - `bigclaw-go/internal/uireview/uireview.go`
    - `bigclaw-go/internal/uireview/builder.go`
    - `bigclaw-go/internal/uireview/render.go`
  - Evidence:
    - `bigclaw-go/internal/uireview/uireview_test.go`
    - `docs/issue-plan.md`
    - `reports/OPE-128-validation.md`

## Why This Sweep Is Now Safe

`reports/BIG-GO-948-validation.md` previously deferred these tests until broader Go-native contract surfaces existed. Those owners now exist in-repo:

- control-center behavior is represented by the Go control plane and v2 operations endpoints.
- operations contract coverage is split across Go dashboard/run-detail and execution contract surfaces.
- UI review coverage is represented by a Go-native review-pack builder, auditor, and report renderer.

## Validation Commands And Results

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
  Result: no output; repository-wide Python file count remained `0`.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1365LegacyTestContractSweepB(ManifestMatchesDeferredLegacyTests|ReplacementPathsExist|LaneReportCapturesReplacementState)$'`
  Result: `ok  	bigclaw-go/internal/regression	0.527s`
