# BIG-GO-1551 Workpad

## Plan

1. Reconfirm the current repository-wide and `src/bigclaw` physical Python file
   counts from the checked-out baseline and capture the exact delta available in
   this lane.
2. Record the blocker reality for this refill slice with ticket-specific
   artifacts: a lane report, validation report, and status metadata that
   include the exact current counts plus the exact historical `src/bigclaw`
   delete ledger from repository history.
3. Add a targeted Go regression guard that locks the `src/bigclaw` zero-Python
   baseline and the ticket report content.
4. Run targeted validation, record exact commands and results, then commit and
   push `BIG-GO-1551`.

## Acceptance

- The lane records repository-wide `.py` counts before and after the change.
- The lane records `src/bigclaw` `.py` counts before and after the change.
- The lane records the exact before-after delta available in this checkout.
- The lane records exact `src/bigclaw` removed-file evidence from repository
  history.
- The lane states whether the ticket is blocked by an already-zero baseline.
- The lane includes a targeted Go regression guard and recorded validation
  commands/results.
- The change is committed and pushed on `BIG-GO-1551`.

## Validation

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find src/bigclaw -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1551(RepositoryHasNoPythonFiles|SrcBigclawDirectoryStaysPythonFree|HistoricalDeletedFileEvidenceIsRecorded|LaneReportCapturesCurrentDeltaAndBlocker)$'`
