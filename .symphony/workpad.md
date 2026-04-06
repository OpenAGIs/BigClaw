# BIG-GO-1552

## Plan

1. Capture the exact baseline repository `.py` count and the exact remaining
   `tests/**/*.py` inventory on top of `origin/BIG-GO-1545`.
2. Physically delete the in-scope test `.py` files from disk and record the
   removed-file ledger plus the repository-wide before/after delta.
3. Add lane-scoped regression coverage and a checked-in evidence report so this
   refill slice proves the deletion set and the count drop.
4. Run targeted validation, then commit and push `BIG-GO-1552`.

## Acceptance

- Remaining `tests/**/*.py` files are physically removed from disk.
- The repository `.py` file count is lower after the change.
- Exact before/after counts are recorded.
- Exact removed-file evidence is recorded.
- Targeted validation commands and results are recorded.
- Changes remain scoped to the test-file deletion refill lane.
- The branch is committed and pushed to `origin/BIG-GO-1552`.

## Validation

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find tests -type f -name '*.py' | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1552(RepositoryPythonCountDrop|TestsPythonFilesDeleted|LaneReportCapturesExactCounts)$'`
