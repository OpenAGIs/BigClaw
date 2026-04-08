# BIG-GO-102 Residual Tests Python Sweep K

## Scope

`BIG-GO-102` closes the residual Python-test gap left after the broader Go-only
test sweep. `BIG-GO-1577` documented five Python tests that only exercised
retired Python modules, but it did not add dedicated regression coverage for
those dependent test paths themselves.

Covered residual Python tests:

- `tests/test_cost_control.py`
- `tests/test_mapping.py`
- `tests/test_repo_board.py`
- `tests/test_repo_collaboration.py`
- `tests/test_roadmap.py`

## Go Or Native Replacement Surfaces

- `bigclaw-go/internal/costcontrol/controller_test.go`
- `bigclaw-go/internal/intake/mapping_test.go`
- `bigclaw-go/internal/repo/board.go`
- `bigclaw-go/internal/collaboration/thread_test.go`
- `bigclaw-go/internal/regression/roadmap_contract_test.go`
- `bigclaw-go/internal/planning/planning_test.go`

## Sweep Result

- Added a dedicated regression guard for the five residual Python tests that
  were named in the `BIG-GO-1577` lane report but were only indirectly covered
  before this issue.
- Bound each retired Python test to the Go/native surface that now owns the
  same behavior so future regressions cannot silently reintroduce the deleted
  Python test files.
- Kept the change scoped to regression/report coverage only; no runtime or
  product behavior changed.

## Validation Commands And Results

- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO102(ResidualPythonTestsStayAbsent|ReplacementSurfacesRemainAvailable|LaneReportCapturesSweepState)$'`
  Result: `ok  	bigclaw-go/internal/regression	0.140s`
