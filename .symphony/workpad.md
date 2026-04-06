# BIG-GO-1554 Workpad

## Plan

1. Reconfirm the repository-wide physical Python file baseline and the focused
   `scripts` / `scripts/ops` wrapper surface for this refill lane.
2. Add lane-scoped reporting artifacts that record the exact before/after
   counts, the exact deleted-file ledger, and the focused residual scan
   evidence for `scripts` / `scripts/ops`.
3. Add focused regression coverage so the repository and the `scripts` /
   `scripts/ops` wrapper surface stay Python-free while the active Go/native
   replacements remain available.
4. Run targeted validation, record the exact commands and outcomes, then
   commit and push the issue branch.

## Acceptance

- The lane records repository-wide `.py` counts before and after the change.
- The lane records the focused `scripts` / `scripts/ops` `.py` counts before
  and after the change.
- The lane includes an exact deleted-file ledger, even if it is empty because
  the baseline is already Python-free.
- The lane names the active Go/native replacement paths for the retired
  `scripts` / `scripts/ops` wrapper surface.
- Exact validation commands and outcomes are recorded in repo-native artifacts.
- The change is committed and pushed on `BIG-GO-1554`.

## Validation

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find scripts scripts/ops -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1554(RepositoryHasNoPythonFiles|ScriptsOpsWrapperSurfaceStaysPythonFree|GoReplacementPathsRemainAvailable|LaneReportCapturesExactCountDeltaAndLedger)$'`
