# BIG-GO-1586 Workpad

## Context
- Issue: `BIG-GO-1586`
- Title: `Strict bucket lane 1586: bigclaw-go/scripts/benchmark/*.py bucket`
- Goal: keep `bigclaw-go/scripts/benchmark` physically free of Python assets and land lane-specific regression evidence with an issue commit.
- Current repo state on entry: `bigclaw-go/scripts/benchmark` contains only `run_suite.sh`; no `.py` files are present in that bucket.

## Scope
- `.symphony/workpad.md`
- `bigclaw-go/internal/regression/big_go_1586_zero_python_guard_test.go`
- `bigclaw-go/docs/reports/big-go-1586-python-asset-sweep.md`
- `reports/BIG-GO-1586-status.json`
- `reports/BIG-GO-1586-validation.md`

## Plan
1. Replace the stale workpad with `BIG-GO-1586` execution details before any code edits.
2. Add a benchmark-bucket regression guard that proves the repository remains Python-free, the retired benchmark Python helpers stay absent, and the surviving native benchmark entrypoints remain available.
3. Add issue-scoped lane report, status, and validation artifacts that record the benchmark bucket inventory and exact verification commands/results.
4. Run targeted inventory and regression commands, then commit and push the lane to `origin/main`.

## Acceptance
- `BIG-GO-1586` has an issue-specific workpad, regression guard, lane report, status artifact, and validation report.
- `bigclaw-go/scripts/benchmark` remains at `0` physical `.py` files.
- Retired benchmark Python helpers stay absent and the remaining benchmark native entrypoints stay available.
- Validation artifacts record the exact commands and exact results used for this lane.
- A scoped lane commit is pushed to the remote branch.

## Validation
- `find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`
- `find bigclaw-go/scripts/benchmark -type f -name '*.py' -print | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1586(RepositoryHasNoPythonFiles|BenchmarkBucketStaysPythonFree|RetiredBenchmarkPythonHelpersRemainAbsent|BenchmarkReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
