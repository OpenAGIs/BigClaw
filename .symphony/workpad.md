# BIG-GO-102 Workpad

## Context
- Issue: `BIG-GO-102`
- Title: `Residual tests Python sweep K`
- Goal: close the residual Python-test sweep gap left by the broader Go-only migration by guarding the remaining retired Python tests that still lack dedicated regression coverage.

## Scope
- `bigclaw-go/internal/regression`
- `bigclaw-go/docs/reports`
- Residual Python test paths called out as direct dependencies of retired Python modules:
  - `tests/test_cost_control.py`
  - `tests/test_mapping.py`
  - `tests/test_repo_board.py`
  - `tests/test_repo_collaboration.py`
  - `tests/test_roadmap.py`

## Plan
1. Inspect the existing tranche and `BIG-GO-1577` regression/report pattern to confirm the uncovered residual-test gap.
2. Add a narrowly scoped regression guard that asserts the residual Python tests stay absent and the Go/native replacement surfaces remain present.
3. Add an issue report for `BIG-GO-102` that captures the covered paths, replacement surfaces, and exact validation commands/results.
4. Run targeted regression tests, record exact commands and outcomes, then commit and push the issue branch.

## Acceptance
- The five residual Python test paths above are explicitly guarded by regression coverage.
- The new report documents the residual-test sweep scope and the Go/native replacement surfaces that own the behavior now.
- Validation records exact commands and observed results.
- Changes stay scoped to the residual Python-test sweep gap only.

## Validation
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO102(ResidualPythonTestsStayAbsent|ReplacementSurfacesRemainAvailable|LaneReportCapturesSweepState)$'`
  - Result: `ok  	bigclaw-go/internal/regression	0.140s`
