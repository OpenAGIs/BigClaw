# BIG-GO-22 Workpad

## Plan

1. Materialize the minimum local checkout needed to modify and validate the
   Go regression package for this lane.
2. Add an issue-scoped zero-Python regression guard for the normalized
   repository state, centered on the retired `src/bigclaw` batch-D surface.
3. Add repo-visible evidence artifacts for the lane:
   - `bigclaw-go/docs/reports/big-go-22-python-asset-sweep.md`
   - `reports/BIG-GO-22-validation.md`
   - `reports/BIG-GO-22-status.json`
4. Run targeted validation, record exact commands plus results, then commit and
   push the branch.

## Acceptance

- `BIG-GO-22` adds issue-scoped regression coverage proving the repository
  remains physically Python-free.
- The lane records that the historical `src/bigclaw` batch-D slice is already
  absent in the current normalized checkout and that the priority residual
  directories remain Python-free.
- Validation artifacts capture the exact commands and outcomes for this lane.

## Validation

- `find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`
- `find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go/internal/regression && go test -count=1 repo_helpers_test.go big_go_22_zero_python_guard_test.go`
- `python3 -m json.tool reports/BIG-GO-22-status.json`
