# BIG-GO-2 Workpad

## Plan

1. Audit the current repository Python baseline and inspect existing Go
regression/report conventions for prior `tests/*.py` sweep lanes so this
issue stays scoped to test-residual coverage.
2. Add `BIG-GO-2` regression coverage that locks the highest-value
test-related residual directories at zero Python files and asserts the
retained Go/native replacement assets remain available.
3. Add the matching lane report plus `reports/BIG-GO-2-{validation,status}`
artifacts, run targeted validation, record exact commands and results, then
commit and push the scoped change set.

## Acceptance

- `BIG-GO-2` has issue-specific regression coverage for the broad
  `tests/*.py` cleanup lane.
- The guard enforces that the selected test-residual directories remain free
  of `.py` files.
- The lane report and `reports/BIG-GO-2-{validation,status}` artifacts record
  the zero-Python inventory, the retained replacement assets, and the exact
  validation commands/results.
- The resulting change is committed and pushed to the remote branch.

## Validation

- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-2 -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-2/tests /Users/openagi/code/bigclaw-workspaces/BIG-GO-2/bigclaw-go/internal/regression /Users/openagi/code/bigclaw-workspaces/BIG-GO-2/bigclaw-go/cmd/bigclawctl /Users/openagi/code/bigclaw-workspaces/BIG-GO-2/bigclaw-go/internal/evaluation /Users/openagi/code/bigclaw-workspaces/BIG-GO-2/bigclaw-go/internal/workflow -type f -name '*.py' 2>/dev/null | sort`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-2/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO2(RepositoryHasNoPythonFiles|PriorityResidualTestDirectoriesStayPythonFree|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
