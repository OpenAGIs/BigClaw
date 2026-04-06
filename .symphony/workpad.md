# BIG-GO-1536 Workpad

## Plan

1. Reconfirm the repository-wide physical Python file baseline and the focused
   `workspace/bootstrap/planning` residual area from the current `main`
   checkout.
2. Add issue-scoped reporting artifacts that capture the exact before/after
   counts, the deleted-file ledger, and the validation evidence for this lane.
3. Add a focused regression guard that keeps the
   `workspace/bootstrap/planning` residual area Python-free and asserts the
   lane report contents.
4. Run targeted validation, record the exact commands and results in repo
   artifacts, then commit and push `BIG-GO-1536`.

## Acceptance

- The lane records repository-wide `.py` counts before and after the change.
- The lane records the focused `workspace/bootstrap/planning` residual scan.
- The lane includes an exact deleted-file ledger, even if the ledger is empty
  because the checkout is already Python-free.
- The lane names the active Go/native replacement paths for the retired
  `workspace/bootstrap/planning` surface.
- Exact validation commands and outcomes are recorded in repo-native artifacts.
- The change is committed and pushed on `BIG-GO-1536`.

## Validation

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find workspace bootstrap planning bigclaw-go/internal/bootstrap bigclaw-go/internal/planning -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1536(RepositoryHasNoPythonFiles|WorkspaceBootstrapPlanningResidualAreaStaysPythonFree|GoReplacementPathsRemainAvailable|LaneReportCapturesExactLedger)$'`
