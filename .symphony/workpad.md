# BIG-GO-11 Workpad

## Context
- Issue: `BIG-GO-11`
- Title: `Sweep remaining Python under src/bigclaw batch C`
- Goal: keep the repository on its current Go-only baseline and add issue-specific regression evidence for the already-empty `src/bigclaw` Python surface.
- Current repo state on entry: repository-wide physical `.py` inventory is `0`, including `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.

## Scope
- `.symphony/workpad.md`
- `bigclaw-go/internal/regression/big_go_11_zero_python_guard_test.go`
- `bigclaw-go/docs/reports/big-go-11-python-asset-sweep.md`
- `reports/BIG-GO-11-status.json`
- `reports/BIG-GO-11-validation.md`

## Plan
1. Replace the stale workpad with issue-specific plan, acceptance, and validation targets before code edits.
2. Add a lane-specific regression guard and sweep report that document the zero-Python baseline and the active Go/native replacement paths.
3. Run targeted inventory and regression commands, record exact commands and results, then commit and push the issue branch to the remote.

## Acceptance
- `BIG-GO-11` has an issue-specific workpad, regression guard, sweep report, validation report, and status artifact.
- The regression guard verifies repository-wide Python file count `0`, Python-free priority residual directories, required Go/native replacement paths, and the lane report content.
- Validation records exact commands and exact results for repository inventory, priority directory inventory, and targeted regression coverage.
- Changes remain scoped to `BIG-GO-11` Python-sweep evidence only.

## Validation
- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO11(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
