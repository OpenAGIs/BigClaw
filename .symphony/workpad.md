# BIG-GO-1568 Workpad

## Plan

- Confirm the live repository baseline for `*.py` files and verify whether this
  lane can still delete physical Python files.
- Audit the remaining `scripts/ops` migration-helper surface and its Go/native
  replacements to select a narrow, unblocked tranche.
- Add issue-specific regression coverage and an exact-ledger report proving the
  `scripts/ops` helper tranche remains Python-free and backed by Go/native
  entrypoints.
- Run targeted validation, then commit and push the lane branch.

## Acceptance

- `find . -name '*.py' | wc -l` is rechecked and recorded for the current
  branch baseline.
- Because the fetched `main` baseline is already at `0`, this lane satisfies
  acceptance through exact Go/native replacement evidence in git instead of a
  physical Python-file count drop.
- The landed evidence stays scoped to the `scripts/ops` migration-helper
  tranche and does not reopen bookkeeping-only work outside that surface.
- The branch contains issue-specific artifacts for `BIG-GO-1568`, targeted test
  execution evidence, and a pushed commit on `origin/BIG-GO-1568`.

## Validation

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find scripts/ops bigclaw-go/cmd/bigclawctl bigclaw-go/internal/refill -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1568(RepositoryHasNoPythonFiles|OpsMigrationHelperResidualAreaStaysPythonFree|GoReplacementPathsRemainAvailable|LaneReportCapturesExactLedger)$'`
