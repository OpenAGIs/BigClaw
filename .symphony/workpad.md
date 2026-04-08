# BIG-GO-110 Workpad

## Context
- Issue: `BIG-GO-110`
- Goal: keep pressure on the practical Go-only repo state by documenting and guarding a repository-wide Python-file budget of `<=1`, while proving the current checkout still sits at `0`.
- Current repo state on entry: repository-wide physical `.py` file inventory is already `0`, including the priority residual directories `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.

## Scope
- `.symphony/workpad.md`
- `bigclaw-go/internal/regression/big_go_110_python_budget_guard_test.go`
- `bigclaw-go/docs/reports/big-go-110-python-budget-sweep.md`
- `reports/BIG-GO-110-status.json`
- `reports/BIG-GO-110-validation.md`

## Plan
1. Replace the stale workpad with issue-specific plan, acceptance criteria, and validation commands before any repo edits.
2. Add a lane-specific regression guard that enforces the practical Python budget, priority-directory zero baseline, active Go/native replacement paths, and issue report content.
3. Add issue-scoped sweep and validation artifacts that record the current `0`-file baseline and frame it as comfortably inside the `<=1` convergence target.
4. Run targeted inventory and regression commands, record exact commands and results, then commit and push the lane branch to `origin/main`.

## Acceptance
- `BIG-GO-110` has issue-specific workpad, regression guard, lane report, validation report, and status metadata.
- The regression guard enforces repository-wide Python file count `<=1`, proves the current branch stays at `0`, verifies the priority residual directories remain Python-free, confirms the active Go/native replacement paths exist, and checks the lane report contains the convergence budget and validation commands.
- Validation records the exact inventory and targeted regression commands together with their exact results.
- Changes remain scoped to this convergence lane and do not broaden into unrelated Go-mainline work.

## Validation
- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO110(RepositoryPythonFileBudgetStaysWithinOne|RepositoryCurrentlyHasZeroPythonFiles|PriorityResidualDirectoriesStayPythonFree|GoReplacementPathsRemainAvailable|LaneReportCapturesBudgetAndSweepState)$'`
