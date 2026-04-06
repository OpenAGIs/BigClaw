# BIG-GO-1559 Workpad

## Plan

1. Reconfirm the repository-wide physical Python baseline and verify whether any
   residual directory still contains `.py` files in the current checkout.
2. Record exact before/after counts, the removed-file ledger, and the
   repository-reality blocker for the largest residual-directory sweep in
   issue-scoped artifacts.
3. Add a focused regression guard that keeps the repository Python-free and
   verifies the `BIG-GO-1559` report captures the exact sweep state.
4. Run targeted validation, record exact commands and results, then commit and
   push `BIG-GO-1559`.

## Acceptance

- The lane records repository-wide physical `.py` counts before and after the
  change.
- The lane records the largest residual-directory scan attempted by this lane.
- The lane records the exact deleted-file ledger, even if no deletions were
  possible because the checkout was already Python-free.
- The lane records the repository-reality blocker that prevents a drop below
  the current baseline.
- Exact validation commands and outcomes are captured in repo-native artifacts.
- The change is committed and pushed on `BIG-GO-1559`.

## Validation

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find src tests scripts workspace bootstrap planning bigclaw-go/scripts bigclaw-go/internal -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1559(RepositoryHasNoPythonFiles|LargestResidualDirectoriesStayPythonFree|GoReplacementPathsRemainAvailable|LaneReportCapturesExactSweepState)$'`

## GitHub

- Branch to push: `origin/BIG-GO-1559`
