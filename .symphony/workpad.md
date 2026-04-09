# BIG-GO-191 Workpad

## Plan

1. Confirm the current repository Python baseline and inspect the residual
priority directories relevant to this lane: `src/bigclaw`, `tests`,
`scripts`, and `bigclaw-go/scripts`.
2. Add lane-specific regression coverage for `BIG-GO-191` that locks those
priority directories at zero Python files while asserting the retained Go and
native replacement entrypoints still exist.
3. Add the matching lane report plus `reports/BIG-GO-191-validation.md` and
`reports/BIG-GO-191-status.json`, run targeted validation, record exact
commands and results, then commit and push the scoped change set.

## Acceptance

- `BIG-GO-191` has lane-specific regression coverage for the residual
  `src/bigclaw` Python sweep.
- The guard enforces that `src/bigclaw`, `tests`, `scripts`, and
  `bigclaw-go/scripts` remain Python-free.
- The lane report and issue artifacts document the zero-Python inventory,
  retained Go/native replacement paths, and the exact validation
  commands/results.
- The resulting change is committed and pushed to the remote branch.

## Validation

- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-191 -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-191/src/bigclaw /Users/openagi/code/bigclaw-workspaces/BIG-GO-191/tests /Users/openagi/code/bigclaw-workspaces/BIG-GO-191/scripts /Users/openagi/code/bigclaw-workspaces/BIG-GO-191/bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-191/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO191(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
