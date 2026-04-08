# BIG-GO-1586 Python Asset Sweep

## Scope

`BIG-GO-1586` records the remaining physical Python asset inventory for the `bigclaw-go/scripts/benchmark/*.py` bucket and confirms the native benchmark replacement surface remains intact.

## Remaining Python Inventory

Repository-wide Python file count: `0`.

- `bigclaw-go/scripts/benchmark`: `0` Python files

This workspace therefore lands as a regression-prevention lane: the target bucket was already physically clean on entry, and this change locks that state in with issue-specific tests and validation evidence.

## Retired Benchmark Python Helpers

The retired benchmark Python helpers that must remain absent are:

- `bigclaw-go/scripts/benchmark/capacity_certification.py`
- `bigclaw-go/scripts/benchmark/capacity_certification_test.py`
- `bigclaw-go/scripts/benchmark/run_matrix.py`
- `bigclaw-go/scripts/benchmark/soak_local.py`

## Native Replacement Paths

The benchmark bucket is now represented by native entrypoints and checked-in benchmark artifacts:

- `bigclaw-go/scripts/benchmark/run_suite.sh`
- `bigclaw-go/cmd/bigclawctl/automation_benchmark_commands.go`
- `bigclaw-go/cmd/bigclawctl/automation_commands_test.go`
- `bigclaw-go/internal/queue/benchmark_test.go`
- `bigclaw-go/internal/scheduler/benchmark_test.go`
- `bigclaw-go/docs/reports/benchmark-report.md`
- `bigclaw-go/docs/reports/benchmark-matrix-report.json`
- `bigclaw-go/docs/reports/long-duration-soak-report.md`
- `bigclaw-go/docs/reports/soak-local-50x8.json`

## Validation Commands And Results

- `find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`
  Result: no output; repository-wide Python file count remained `0`.
- `find bigclaw-go/scripts/benchmark -type f -name '*.py' -print | sort`
  Result: no output; the benchmark bucket remained Python-free.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1586(RepositoryHasNoPythonFiles|BenchmarkBucketStaysPythonFree|RetiredBenchmarkPythonHelpersRemainAbsent|BenchmarkReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
  Result: `ok  	bigclaw-go/internal/regression	4.124s`
