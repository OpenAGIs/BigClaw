# BIG-GO-1535 Workpad

## Plan

1. Reconfirm the repository-wide physical Python file baseline and the focused
   reporting / observability residual area, including the Go replacement
   surfaces that now own that behavior.
2. Add lane-scoped reporting artifacts that record the exact before/after
   counts, the exact deleted-file ledger, and the validation evidence for this
   refill slice.
3. Add focused regression coverage so the repository and the
   reporting / observability residual area stay Python-free.
4. Run targeted validation, record the exact commands and results in checked-in
   artifacts, then commit and push the issue branch.

## Acceptance

- The lane records repository-wide `.py` counts before and after the change.
- The lane records the focused reporting / observability residual scan.
- The lane includes an exact deleted-file ledger, even if the ledger is empty
  because the baseline is already Python-free.
- The lane names the active Go/native replacement paths for the retired
  reporting / observability surface.
- Exact validation commands and outcomes are recorded in repo-native artifacts.
- The change is committed and pushed on `BIG-GO-1535`.

## Validation

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find src tests scripts bigclaw-go/internal/observability bigclaw-go/internal/reporting -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1535(RepositoryHasNoPythonFiles|ReportingObservabilityResidualAreaStaysPythonFree|GoReplacementPathsRemainAvailable|LaneReportCapturesExactLedger)$'`
