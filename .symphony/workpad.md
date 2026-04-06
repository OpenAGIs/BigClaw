# BIG-GO-1531 Workpad

## Plan

1. Reconfirm the repository-wide physical Python baseline and the targeted
   `src/bigclaw` residual surface named in this issue.
2. Record lane-scoped before/after counts, the exact deleted-file ledger, and
   explicit absent-file evidence for representative historical `src/bigclaw`
   modules in a checked-in report.
3. Add focused regression coverage so the repository stays Python-free and the
   `src/bigclaw` surface remains physically absent.
4. Run targeted validation, record the exact commands and outcomes, then commit
   and push `BIG-GO-1531`.

## Acceptance

- The lane records repository-wide `.py` counts before and after the change.
- The lane records the focused `src/bigclaw` physical `.py` count before and
  after the lane.
- The lane includes an exact deleted-file ledger, even if the ledger is empty
  because this checkout is already Python-free.
- The lane includes exact absent-file evidence for representative historical
  `src/bigclaw` modules.
- The lane names the active Go/native replacement paths for the retired
  `src/bigclaw` surface.
- Exact validation commands and outcomes are recorded in repo-native artifacts.

## Validation

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find src/bigclaw -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1531(RepositoryHasNoPythonFiles|SrcBigclawSurfaceStaysPythonFree|RepresentativeHistoricalSrcBigclawFilesRemainAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesExactEvidence)$'`
