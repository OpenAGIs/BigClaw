# BIG-GO-1557 Workpad

## Plan

1. Reconfirm the repository-wide physical Python-file baseline on the active
   branch and record the exact before/after counts plus the deleted-file ledger
   for this lane.
2. Add lane-scoped reporting artifacts and a regression guard test that lock in
   the repo-wide zero-Python state and capture the stubborn-sweep evidence.
3. Run targeted validation, record the exact commands and outputs, then commit
   and push the issue branch.

## Acceptance

- The lane records repository-wide physical `.py` counts before and after the
  change.
- The lane records an exact deleted-file ledger for `BIG-GO-1557`.
- The lane names the active Go/native replacement paths that cover the removed
  Python-era repo surface.
- Exact validation commands and outcomes are recorded in checked-in artifacts.
- The work is committed and pushed on `BIG-GO-1557`.

## Validation

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find .github docs scripts bigclaw-go -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1557(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|GoReplacementPathsRemainAvailable|LaneReportCapturesExactLedger)$'`
