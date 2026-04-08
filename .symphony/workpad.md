# BIG-GO-1590 Workpad

## Plan

1. Confirm the current repository-wide physical Python baseline and identify the
   repo surfaces that still represent the "physical reduction bucket" for this
   lane.
2. Add only `BIG-GO-1590`-scoped artifacts: a regression guard and a lane report
   that record the empty bucket state plus the retained Go/native replacement
   surfaces.
3. Run targeted validation, capture exact commands and results in repo artifacts,
   then commit and push the lane branch.

## Acceptance

- `BIG-GO-1590` has issue-specific artifacts for the repo-wide physical
  reduction bucket.
- Regression coverage fails if physical `.py` files reappear in the repository
  or in the lane's focused residual directories.
- The lane report records exact before/after counts, deleted-file evidence, the
  retained replacement surfaces, and the exact validation commands/results.
- The resulting change is committed and pushed to the remote branch.

## Validation

- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1590 -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1590/docs /Users/openagi/code/bigclaw-workspaces/BIG-GO-1590/scripts /Users/openagi/code/bigclaw-workspaces/BIG-GO-1590/tests /Users/openagi/code/bigclaw-workspaces/BIG-GO-1590/bigclaw-go/internal /Users/openagi/code/bigclaw-workspaces/BIG-GO-1590/bigclaw-go/docs/reports /Users/openagi/code/bigclaw-workspaces/BIG-GO-1590/reports -type f -name '*.py' 2>/dev/null | sort`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1590/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1590(RepositoryHasNoPythonFiles|RepoWidePhysicalReductionBucketStaysPythonFree|GoReplacementPathsRemainAvailable|LaneReportCapturesExactLedger)$'`

## Execution Notes

- 2026-04-09: Initial repository scan found no physical `.py` files anywhere in
  this checkout, so the actionable lane shape is regression hardening plus exact
  zero-count documentation for the repo-wide physical reduction bucket.
- 2026-04-09: `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1590 -path '*/.git' -prune -o -name '*.py' -type f -print | sort` produced no output.
- 2026-04-09: `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1590/docs /Users/openagi/code/bigclaw-workspaces/BIG-GO-1590/scripts /Users/openagi/code/bigclaw-workspaces/BIG-GO-1590/tests /Users/openagi/code/bigclaw-workspaces/BIG-GO-1590/bigclaw-go/internal /Users/openagi/code/bigclaw-workspaces/BIG-GO-1590/bigclaw-go/docs/reports /Users/openagi/code/bigclaw-workspaces/BIG-GO-1590/reports -type f -name '*.py' 2>/dev/null | sort` produced no output.
- 2026-04-09: `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1590/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1590(RepositoryHasNoPythonFiles|RepoWidePhysicalReductionBucketStaysPythonFree|GoReplacementPathsRemainAvailable|LaneReportCapturesExactLedger)$'` returned `ok   bigclaw-go/internal/regression 0.150s`.
