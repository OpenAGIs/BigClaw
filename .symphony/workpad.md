# BIG-GO-1 Workpad

## Context
- Issue: `BIG-GO-1`
- Goal: close out the residual `src/bigclaw` Python sweep by validating the current branch state and locking in a regression guard for the now Python-free tree.
- Current repo state on entry: the assigned workspace bootstrapped into a partial promisor checkout, so the implementation is limited to issue-scoped audit artifacts and a targeted Go regression guard on top of `main`.

## Plan
1. Verify the live tree no longer contains physical Python files, with explicit focus on `src/bigclaw` and the historical residual directories.
2. Add a `BIG-GO-1` regression guard in Go so future changes cannot silently reintroduce Python files under `src/bigclaw` or the broader repository.
3. Record the closeout evidence and exact validation commands in a lane-specific report.
4. Run the targeted inventory commands and Go regression test.
5. Commit and push the issue branch.

## Acceptance
- Repository-wide physical `.py` files remain at `0`.
- `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts` remain free of physical Python files.
- The Go/native replacement paths for the removed Python surface remain present.
- Exact validation commands and outcomes are recorded for this issue.
- The diff stays scoped to `BIG-GO-1` audit and regression-guard artifacts.

## Validation
- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
