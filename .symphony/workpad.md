# BIG-GO-1555 Workpad

## Plan

1. Reconfirm the repository-wide physical Python baseline and the reporting /
   observability residual surface that this lane owns.
2. Add lane-scoped artifacts that record exact before/after counts, the exact
   removed-file ledger, and the active Go replacement paths for the retired
   reporting / observability Python surface.
3. Add focused regression coverage so the reporting / observability Python
   surface stays absent and the lane report keeps the exact ledger evidence.
4. Run targeted validation, record the exact commands and results, then commit
   and push `BIG-GO-1555`.

## Acceptance

- The lane records repository-wide `.py` counts before and after the change.
- The lane records the focused reporting / observability Python residual scan.
- The lane includes an exact deleted-file ledger, even if it is empty because
  the checkout is already Python-free for this surface.
- The lane names the active Go replacement paths for the retired reporting /
  observability surface.
- Exact validation commands and outcomes are recorded in checked-in artifacts.
- The change is committed and pushed on `BIG-GO-1555`.

## Validation

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find src bigclaw-go/internal/observability bigclaw-go/internal/reporting -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1555(RepositoryHasNoPythonFiles|ReportingObservabilityResidualSurfaceStaysPythonFree|GoReplacementPathsRemainAvailable|LaneReportCapturesExactLedger)$'`
