# BIG-GO-1368 Benchmark Python Replacement

## Scope

Go-only refill lane `BIG-GO-1368` records the concrete Go/native owners for the
retired benchmark Python scripts under `bigclaw-go/scripts/benchmark`.

## Remaining Python Inventory

Repository-wide Python file count: `0`.

This checkout therefore cannot satisfy the lane by deleting new `.py` files.
The acceptance path is concrete benchmark replacement evidence pinned in git.

## Benchmark Replacement Registry

The benchmark sweep for this lane is captured in
`bigclaw-go/internal/migration/benchmark_script_replacements.go`.

- `bigclaw-go/scripts/benchmark/soak_local.py`
  Replaced by `bigclaw-go/cmd/bigclawctl/automation_commands.go`,
  `bigclaw-go/cmd/bigclawctl/automation_commands_test.go`,
  `bigclaw-go/scripts/benchmark/run_suite.sh`,
  `bigclaw-go/docs/benchmark-plan.md`, and
  `bigclaw-go/docs/reports/long-duration-soak-report.md`.
- `bigclaw-go/scripts/benchmark/run_matrix.py`
  Replaced by `bigclaw-go/cmd/bigclawctl/automation_benchmark_commands.go`,
  `bigclaw-go/cmd/bigclawctl/automation_commands_test.go`,
  `bigclaw-go/scripts/benchmark/run_suite.sh`,
  `bigclaw-go/docs/benchmark-plan.md`, and
  `bigclaw-go/docs/reports/benchmark-matrix-report.json`.
- `bigclaw-go/scripts/benchmark/capacity_certification.py`
  Replaced by `bigclaw-go/cmd/bigclawctl/automation_benchmark_commands.go`,
  `bigclaw-go/cmd/bigclawctl/automation_commands_test.go`,
  `bigclaw-go/docs/go-cli-script-migration.md`,
  `bigclaw-go/docs/reports/capacity-certification-matrix.json`, and
  `bigclaw-go/docs/reports/capacity-certification-report.md`.
- `bigclaw-go/scripts/benchmark/capacity_certification_test.py`
  Replaced by `bigclaw-go/cmd/bigclawctl/automation_commands_test.go`,
  `bigclaw-go/internal/regression/big_go_1160_script_migration_test.go`, and
  `bigclaw-go/docs/go-cli-script-migration.md`.

## Validation Commands And Results

- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1368 -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
  Result: no output; repository-wide Python file count remained `0`.
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1368/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1368BenchmarkPythonReplacement(ManifestCapturesBenchmarkLane|ReplacementPathsExist|LaneReportCapturesReplacementState)$'`
  Result: `ok  	bigclaw-go/internal/regression	1.022s`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1368/bigclaw-go && go test -count=1 ./cmd/bigclawctl -run 'TestBenchmarkScriptsStayGoOnly|TestAutomationBenchmarkRunMatrixWritesReport|TestAutomationBenchmarkCapacityCertificationBuildsReport'`
  Result: `ok  	bigclaw-go/cmd/bigclawctl	0.654s`
