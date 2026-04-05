# BIG-GO-1368 Validation

Date: 2026-04-05

## Scope

Issue: `BIG-GO-1368`

Title: `Go-only refill 1368: bigclaw-go/scripts benchmark python replacement sweep`

This lane does not remove in-branch Python files because the checked-out
workspace is already at a repository-wide physical Python count of `0`. Instead,
it lands a concrete Go/native replacement registry for the retired benchmark
Python helpers under `bigclaw-go/scripts/benchmark` and adds targeted
regression coverage around that registry.

## Delivered Artifact

- Go-native replacement registry:
  `bigclaw-go/internal/migration/benchmark_script_replacements.go`
- Lane report:
  `bigclaw-go/docs/reports/big-go-1368-benchmark-python-replacement.md`
- Regression guard:
  `bigclaw-go/internal/regression/big_go_1368_benchmark_python_replacement_test.go`

## Validation Commands

- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1368 -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1368/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1368BenchmarkPythonReplacement(ManifestCapturesBenchmarkLane|ReplacementPathsExist|LaneReportCapturesReplacementState)$'`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1368/bigclaw-go && go test -count=1 ./cmd/bigclawctl -run 'TestBenchmarkScriptsStayGoOnly|TestAutomationBenchmarkRunMatrixWritesReport|TestAutomationBenchmarkCapacityCertificationBuildsReport'`

## Validation Results

### Repository Python inventory

Command:

```bash
find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1368 -path '*/.git' -prune -o -name '*.py' -type f -print | sort
```

Result:

```text
none
```

### Benchmark replacement regression guard

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1368/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1368BenchmarkPythonReplacement(ManifestCapturesBenchmarkLane|ReplacementPathsExist|LaneReportCapturesReplacementState)$'
```

Result:

```text
ok  	bigclaw-go/internal/regression	1.022s
```

### Benchmark CLI coverage

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1368/bigclaw-go && go test -count=1 ./cmd/bigclawctl -run 'TestBenchmarkScriptsStayGoOnly|TestAutomationBenchmarkRunMatrixWritesReport|TestAutomationBenchmarkCapacityCertificationBuildsReport'
```

Result:

```text
ok  	bigclaw-go/cmd/bigclawctl	0.654s
```

## Git

- Branch: `feat/BIG-GO-1368-benchmark-python-replacement`
- Baseline HEAD before lane commit: `81654c01`
- Lane commit details: `a0232c6b feat(bigclaw-go): record benchmark python replacement sweep`
- Final pushed lane commit: `see git rev-parse --short HEAD`
- Push target: `origin/feat/BIG-GO-1368-benchmark-python-replacement`

## Residual Risk

- The branch baseline was already Python-free, so `BIG-GO-1368` proves the
  benchmark-script replacement by landing a Go-native ownership registry rather
  than by numerically reducing the repository `.py` count.
