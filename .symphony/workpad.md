# BIG-GO-157 Workpad

## Context
- Issue: `BIG-GO-157`
- Goal: harden the Go-only baseline for the repo-wide Python reduction sweep U by auditing the remaining high-impact operational and report-heavy directories that historically concentrated Python-adjacent tooling.
- Current repo state on entry: repository-wide physical Python inventory is already `0`.

## Scope
- `.symphony/workpad.md`
- `bigclaw-go/internal/regression/big_go_157_zero_python_guard_test.go`
- `bigclaw-go/docs/reports/big-go-157-python-asset-sweep.md`
- `reports/BIG-GO-157-status.json`
- `reports/BIG-GO-157-validation.md`

## Plan
1. Replace the stale workpad with an issue-specific plan, acceptance criteria, and validation targets before any code edits.
2. Add a lane-specific regression guard that verifies repository-wide zero Python, the standard priority residual directories, and the high-impact operational and report-heavy directories covered by this sweep.
3. Add lane artifacts documenting the audited directories, the retained native replacement paths, and the exact validation commands and results for this checkout.
4. Run targeted inventory and regression commands, record exact commands and exact results, then commit and push the lane commit to the remote tracking branch.

## Acceptance
- `BIG-GO-157` has an issue-specific workpad, regression guard, lane report, validation report, and status artifact.
- The regression guard verifies repository-wide Python file count `0`, keeps the priority residual directories Python-free, and locks down the audited broad-sweep directories for this lane.
- The lane report and validation report record exact commands and exact results for repository inventory, lane-specific directory inventory, and the targeted Go regression run.
- Changes remain scoped to `BIG-GO-157` artifacts only.

## Validation
- `find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`
- `find docs docs/reports reports scripts bigclaw-go/scripts bigclaw-go/docs/reports bigclaw-go/examples -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO157(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|BroadSweepDirectoriesStayPythonFree|NativeReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
