# BIG-GO-137 Workpad

## Context
- Issue: `BIG-GO-137`
- Goal: complete a broad repo Python reduction sweep over the remaining high-impact directories that historically carried helper, reporting, and operational glue.
- Current repo state on entry: the checked-out workspace already reports a repository-wide physical Python file count of `0`.

## Scope
- `.symphony/workpad.md`
- `bigclaw-go/internal/regression/big_go_137_zero_python_guard_test.go`
- `bigclaw-go/docs/reports/big-go-137-python-asset-sweep.md`
- `reports/BIG-GO-137-status.json`
- `reports/BIG-GO-137-validation.md`

## Plan
1. Replace the stale workpad with issue-specific scope, acceptance, and validation targets before code edits.
2. Add a lane-specific regression guard that preserves the repository-wide zero-Python baseline, the priority residual directories, and a broader high-impact auxiliary sweep across docs, reports, ops, and control surfaces.
3. Add a lane report plus status and validation artifacts that document the audited directories, surviving Go/native replacement paths, and the exact validation commands and outputs.
4. Run targeted inventory and regression commands, capture exact results, then commit and push the lane changes to the remote branch.

## Acceptance
- `BIG-GO-137` has an issue-specific workpad, regression guard, lane report, status artifact, and validation report.
- The regression guard verifies repository-wide Python file count `0`, keeps the priority residual directories Python-free, and locks down the broader high-impact auxiliary sweep directories chosen for this lane.
- Validation records exact commands and exact results for repository inventory, auxiliary directory inventory, and the targeted regression guard.
- Changes remain scoped to `BIG-GO-137` artifacts only.

## Validation
- `find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`
- `find .github .githooks .symphony docs docs/reports reports scripts/ops bigclaw-go/docs bigclaw-go/docs/reports bigclaw-go/examples -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO137(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|BroadRepoHighImpactDirectoriesStayPythonFree|GoNativeReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
