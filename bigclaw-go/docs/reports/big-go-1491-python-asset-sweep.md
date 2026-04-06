# BIG-GO-1491 Python Asset Sweep

Date: 2026-04-06

## Summary

`BIG-GO-1491` audited the largest residual sweep targets for physical Python
files in the checked-out repository. The baseline branch was already Go-only,
so there was no remaining `.py` file to delete in-branch.

- Repository-wide Python file count before sweep: `0`
- Repository-wide Python file count after sweep: `0`
- Net physical Python files removed by this lane: `0`

## Residual Directory Counts

- `src/bigclaw`: `0` Python files before, `0` after
- `tests`: `0` Python files before, `0` after
- `scripts`: `0` Python files before, `0` after
- `bigclaw-go/scripts`: `0` Python files before, `0` after

## Deleted File List

- None. The checked-out baseline already had no physical Python files.

## Go Ownership Or Delete Conditions

- `src/bigclaw`: delete condition already satisfied because the directory is
  absent from the baseline checkout.
- `tests`: delete condition already satisfied because the directory is absent
  from the baseline checkout.
- `scripts`: Go/native ownership remains with `scripts/dev_bootstrap.sh`,
  `scripts/ops/bigclawctl`, `scripts/ops/bigclaw-issue`,
  `scripts/ops/bigclaw-panel`, and `scripts/ops/bigclaw-symphony`.
- `bigclaw-go/scripts`: Go/native ownership remains with
  `bigclaw-go/scripts/e2e/run_all.sh`,
  `bigclaw-go/cmd/bigclawctl/main.go`, and
  `bigclaw-go/cmd/bigclawd/main.go`.

## Validation Commands

- `find . -path '*/.git' -prune -o -type f -name '*.py' -print | sed 's#^./##' | sort`
- `find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1491(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|GoOwnershipAndDeleteConditionsRemainDocumented|LaneReportCapturesBeforeAfterCounts)$'`
