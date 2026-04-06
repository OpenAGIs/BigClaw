# BIG-GO-1556 Workpad

## Plan

1. Reconfirm the repository-wide physical Python file baseline and the focused
   `workspace/bootstrap/planning` residual area from the `BIG-GO-1556` branch
   point.
2. Write `BIG-GO-1556`-scoped repo-native artifacts with exact before/after
   counts, the exact removed-file ledger, the active Go/native replacement
   paths, and the baseline blocker note.
3. Add focused regression coverage that keeps the repository and the
   `workspace/bootstrap/planning` residual area Python-free while pinning the
   `BIG-GO-1556` report contents.
4. Run targeted validation, record exact commands and results, then commit and
   push `BIG-GO-1556`.

## Acceptance

- `BIG-GO-1556` work records repository-wide `.py` counts before and after the
  lane.
- `BIG-GO-1556` work records the focused `workspace/bootstrap/planning`
  residual scan.
- `BIG-GO-1556` work includes an exact deleted-file ledger, even when it is
  empty because the baseline is already Python-free.
- `BIG-GO-1556` work names the active Go/native replacement paths for the
  retired `workspace/bootstrap/planning` surface.
- `BIG-GO-1556` work records the blocker that a lower physical `.py` file count
  cannot be achieved from baseline commit `646edf33` because the repository is
  already at `0`.
- Exact validation commands and outcomes are recorded in repo-native artifacts.

## Validation

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find workspace bootstrap planning bigclaw-go/internal/bootstrap bigclaw-go/internal/planning -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1556(RepositoryHasNoPythonFiles|WorkspaceBootstrapPlanningResidualAreaStaysPythonFree|GoReplacementPathsRemainAvailable|LaneReportCapturesExactLedgerAndBlocker)$'`
