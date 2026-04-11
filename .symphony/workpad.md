# BIG-GO-21 Workpad

## Plan

1. Confirm the repository-wide Python baseline and the batch C residual
   `src/bigclaw` surface tied to `workspace_bootstrap_validation.py`.
2. Add lane-specific regression coverage for `BIG-GO-21` that keeps the
   retired batch C Python path absent, verifies the focused sweep surface stays
   Python-free, and asserts the Go replacement artifacts remain available.
3. Add the matching lane report plus `reports/BIG-GO-21-validation.md` and
   `reports/BIG-GO-21-status.json`, run targeted validation, record exact
   commands and results, then commit and push the scoped change set.

## Acceptance

- `BIG-GO-21` has lane-specific regression coverage for batch C of the
  `src/bigclaw` Python sweep.
- The guard enforces that `src/bigclaw` and
  `bigclaw-go/internal/bootstrap` remain Python-free for this issue surface.
- The retired Python path `src/bigclaw/workspace_bootstrap_validation.py`
  remains absent and the Go replacement files remain present.
- The lane report and issue artifacts document the zero-Python inventory,
  retained Go/native replacement paths, and the exact validation
  commands/results.
- The resulting change is committed and pushed to the remote branch.

## Validation

- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-21 -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-21/src/bigclaw /Users/openagi/code/bigclaw-workspaces/BIG-GO-21/bigclaw-go/internal/bootstrap -type f -name '*.py' 2>/dev/null | sort`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-21/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO21(RepositoryHasNoPythonFiles|BatchCSweepSurfaceStaysPythonFree|RetiredBatchCPythonPathRemainsAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$|TestTopLevelModulePurgeTranche3$'`
