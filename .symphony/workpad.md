# BIG-GO-168 Workpad

## Context
- Issue: `BIG-GO-168`
- Title: `Broad repo Python reduction sweep X`
- Goal: keep the repo-wide broad residual surfaces Python-free and record lane-specific regression evidence for the normalized Go-only baseline.
- Current repo state on entry: `find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort` returns no files.

## Scope
- `.symphony/workpad.md`
- `bigclaw-go/internal/regression/big_go_168_zero_python_guard_test.go`
- `bigclaw-go/docs/reports/big-go-168-python-asset-sweep.md`
- `reports/BIG-GO-168-status.json`
- `reports/BIG-GO-168-validation.md`

## Plan
1. Replace the stale workpad with `BIG-GO-168` plan, acceptance criteria, and exact validation commands before editing code or reports.
2. Add a lane-specific regression guard that verifies repository-wide zero Python, the priority residual directories stay Python-free, the broad sweep directories stay Python-free, and the native replacement paths remain available.
3. Add the issue-scoped lane report, status artifact, and validation report documenting the audited directories, exact commands, and exact validation results.
4. Run the targeted inventory and regression commands, update the recorded outputs, then commit and push the lane branch.

## Acceptance
- `BIG-GO-168` has an issue-specific workpad, regression guard, lane report, status artifact, and validation report.
- The repository-wide physical `.py` count remains `0`.
- The audited broad-sweep directories `docs`, `docs/reports`, `reports`, `scripts`, `bigclaw-go/scripts`, `bigclaw-go/docs/reports`, and `bigclaw-go/examples` remain Python-free.
- The regression guard proves the repo stays Python-free and the native Go/shell replacement paths remain available.
- Validation artifacts record the exact commands and exact observed results for this lane.

## Validation
- `find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`
- `find docs docs/reports reports scripts bigclaw-go/scripts bigclaw-go/docs/reports bigclaw-go/examples -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO168(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|BroadSweepDirectoriesStayPythonFree|NativeReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
