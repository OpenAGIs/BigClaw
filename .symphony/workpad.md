# BIG-GO-127 Workpad

## Context
- Issue: `BIG-GO-127`
- Goal: complete broad repo Python reduction sweep O by hardening the already Python-free checkout with lane-specific regression and validation artifacts.
- Current repo state on entry: repository-wide physical `.py` inventory is already `0`, including the main priority directories `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.

## Scope
- `.symphony/workpad.md`
- `bigclaw-go/internal/regression/big_go_127_zero_python_guard_test.go`
- `bigclaw-go/docs/reports/big-go-127-python-asset-sweep.md`
- `reports/BIG-GO-127-status.json`
- `reports/BIG-GO-127-validation.md`

## Plan
1. Replace the stale workpad with issue-specific scope, acceptance, and validation targets before code edits.
2. Add a lane-specific regression guard for repository-wide zero Python, the priority residual directories, and the active Go/native replacement paths that cover this broad sweep.
3. Add a lane report plus status and validation artifacts that document the zero-Python inventory and exact validation evidence for this lane.
4. Run the targeted inventory and regression commands, record exact commands and exact results, then commit and push the lane branch to `origin/main`.

## Acceptance
- `BIG-GO-127` has an issue-specific workpad, regression guard, lane report, status artifact, and validation report.
- The regression guard verifies repository-wide Python file count `0`, keeps `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts` Python-free, and locks in the current Go/native replacement paths.
- Validation records exact commands and exact results for repository inventory, priority-directory inventory, and targeted regression coverage.
- Changes remain scoped to `BIG-GO-127` artifacts only.

## Validation
- `find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`
- `find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO127(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
