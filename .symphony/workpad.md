# BIG-GO-217 Workpad

## Plan

1. Confirm the live repository-wide Python asset inventory and verify that the
   highest-density residual broad-repo surfaces in this checkout are already
   physically Python-free:
   - `reports`
   - `bigclaw-go/docs`
   - `bigclaw-go/docs/reports`
   - `bigclaw-go/internal`
   - `bigclaw-go/internal/regression`
2. Add the lane-scoped evidence bundle for `BIG-GO-217` so this unattended run
   records the zero-Python baseline and the retained Go/native replacement
   surface:
   - `bigclaw-go/internal/regression/big_go_217_zero_python_guard_test.go`
   - `bigclaw-go/docs/reports/big-go-217-python-asset-sweep.md`
   - `reports/BIG-GO-217-validation.md`
   - `reports/BIG-GO-217-status.json`
3. Run the targeted inventory checks and regression test, record exact
   commands and results, then commit and push the issue-scoped changes to
   `origin/main`.

## Acceptance

- The assigned broad repo residual directories are verified Python-free in the
  live checkout, with repo-visible evidence tied to `BIG-GO-217`.
- `BIG-GO-217` adds a Go regression guard covering the repository-wide
  zero-Python baseline, the priority residual directories, and the retained
  Go/native replacement surface for this broad sweep.
- The lane report and validation report record the exact validation commands,
  observed results, and the already-zero baseline caveat.
- The resulting change set is committed and pushed.

## Validation

- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-217 -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-217/reports /Users/openagi/code/bigclaw-workspaces/BIG-GO-217/bigclaw-go/docs /Users/openagi/code/bigclaw-workspaces/BIG-GO-217/bigclaw-go/docs/reports /Users/openagi/code/bigclaw-workspaces/BIG-GO-217/bigclaw-go/internal /Users/openagi/code/bigclaw-workspaces/BIG-GO-217/bigclaw-go/internal/regression -type f -name '*.py' 2>/dev/null | sort`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-217/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO217(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`

## Execution Notes

- 2026-04-11: Initial inspection shows the checkout is already at a
  repository-wide Python file count of `0`.
- 2026-04-11: `BIG-GO-217` therefore hardens the zero-Python baseline for the
  highest-density residual broad-repo directories instead of deleting
  in-branch `.py` files.
