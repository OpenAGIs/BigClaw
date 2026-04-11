# BIG-GO-220 Workpad

## Plan

1. Confirm the current repository-wide Python asset inventory and verify the
   practical Go-only baseline for the checkout.
2. Add the lane-scoped evidence bundle for `BIG-GO-220` so this unattended run
   records the zero-Python baseline and the active Go/native replacement paths:
   - `bigclaw-go/internal/regression/big_go_220_zero_python_guard_test.go`
   - `bigclaw-go/docs/reports/big-go-220-python-asset-sweep.md`
   - `reports/BIG-GO-220-validation.md`
   - `reports/BIG-GO-220-status.json`
3. Run the targeted inventory checks and regression test, then commit and push
   the issue-scoped changes to `origin/main`.

## Acceptance

- The live checkout is verified at a practical Go-only baseline with no
  repository `.py` files present.
- `BIG-GO-220` adds a Go regression guard covering the repository-wide
  zero-Python baseline, the priority residual directories, and the retained
  Go/native replacement surface.
- The lane report, validation report, and status JSON record the exact
  validation commands, observed results, and the already-zero baseline caveat.
- The resulting change set is committed and pushed.

## Validation

- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-220 -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-220/src/bigclaw /Users/openagi/code/bigclaw-workspaces/BIG-GO-220/tests /Users/openagi/code/bigclaw-workspaces/BIG-GO-220/scripts /Users/openagi/code/bigclaw-workspaces/BIG-GO-220/bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-220/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO220(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`

## Execution Notes

- 2026-04-11: Initial inspection shows the checkout is already at a
  repository-wide Python file count of `0`.
- 2026-04-11: `BIG-GO-220` therefore hardens and documents the practical
  Go-only baseline instead of deleting in-branch `.py` files.
