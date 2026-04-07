# BIG-GO-1563 Workpad

## Plan

1. Reconfirm the repository-wide physical Python baseline and the specific
   Python-test residual surfaces relevant to "new unblocked tests deletion
   tranche A", including whether any `.py` test files still exist on disk.
2. Record lane-scoped evidence for the already-zero Python baseline, including
   exact before/after counts, an exact deleted-file ledger, and the Go/native
   replacement paths that now cover tranche A test behavior.
3. Add a focused regression guard that fails if Python files reappear or if the
   tranche A replacement surface/report disappears.
4. Run targeted validation, record the exact commands and outcomes in checked-in
   artifacts, then commit and push `BIG-GO-1563`.

## Acceptance

- The repository-wide physical `.py` file count is recorded before and after
  the lane.
- The lane records the focused tranche A test residual scan and exact
  deleted-file ledger, even if the ledger is empty because the checkout is
  already Python-free.
- The lane names the Go/native replacement evidence that covers tranche A test
  behavior.
- Exact validation commands and outcomes are recorded in repo-native artifacts.
- The change is committed and pushed on `BIG-GO-1563`.

## Validation

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find bigclaw-go/internal/observability bigclaw-go/internal/intake bigclaw-go/internal/planning bigclaw-go/internal/reporting bigclaw-go/internal/orchestrator bigclaw-go/internal/regression -type f \\( -name 'audit_test.go' -o -name 'connector_test.go' -o -name 'planning_test.go' -o -name 'reporting_test.go' -o -name 'loop_test.go' -o -name 'python_test_tranche17_removal_test.go' \\) | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1563(RepositoryHasNoPythonFiles|PythonTestsDirectoryStaysRemoved|GoReplacementPathsRemainAvailable|LaneReportCapturesExactLedger)$'`
