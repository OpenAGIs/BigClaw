# BIG-GO-1587 Workpad

## Context
- Issue: `BIG-GO-1587`
- Title: `Strict bucket lane 1587: bigclaw-go/scripts/migration/*.py bucket`
- Goal: keep the `bigclaw-go/scripts/migration` Python bucket physically absent and record lane-specific regression evidence.
- Current repo state on entry: `bigclaw-go/scripts/migration` does not exist and `find bigclaw-go -type f -name '*.py'` returns no files.

## Scope
- `.symphony/workpad.md`
- `bigclaw-go/internal/regression/big_go_1587_zero_python_guard_test.go`
- `bigclaw-go/docs/reports/big-go-1587-python-asset-sweep.md`
- `reports/BIG-GO-1587-status.json`
- `reports/BIG-GO-1587-validation.md`

## Plan
1. Replace the stale workpad with `BIG-GO-1587` execution details before any code edits.
2. Add a lane-specific regression test that verifies repository-wide zero Python, keeps `bigclaw-go/scripts/migration` absent and Python-free, and ensures the active Go migration replacements remain available.
3. Add issue-scoped report, status, and validation artifacts that capture the current bucket state and the exact validation commands/results.
4. Run targeted inventory and regression commands, record exact outputs, then commit and push the branch.

## Acceptance
- `BIG-GO-1587` has an issue-specific workpad, regression guard, lane report, status artifact, and validation report.
- The physical `.py` count under `bigclaw-go/scripts/migration` remains `0` because the directory is absent and no replacement Python files exist elsewhere under `bigclaw-go`.
- The regression guard proves the repository stays Python-free, the `bigclaw-go/scripts/migration` bucket stays absent/Python-free, and the Go migration replacements remain available.
- Validation artifacts record the exact commands and exact results used for this lane.

## Validation
- `find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`
- `find bigclaw-go/scripts/migration -type f -name '*.py' 2>/dev/null | sort`
- `test ! -d bigclaw-go/scripts/migration`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1587(RepositoryHasNoPythonFiles|MigrationBucketStaysAbsentAndPythonFree|GoMigrationReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
