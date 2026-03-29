# BIG-GO-969 Workpad

## Plan

1. Inventory `bigclaw-go/scripts/benchmark/**` Python files, trace repo references, and classify each file as delete, replace, or keep within this issue scope.
2. Migrate the remaining benchmark matrix and capacity certification workflows from Python into `bigclaw-go` Go-owned automation code so the benchmark directory no longer depends on Python.
3. Remove the Python benchmark scripts once the Go command surface and tests cover their behavior.
4. Update the repo documentation and checked-in references to point at the Go entrypoints and record the benchmark-file disposition for this lane.
5. Run targeted validation for the touched Go command surface, verify the repo Python-file count impact, then commit and push the scoped branch.

## Acceptance

- Produce the explicit file list for this lane under `bigclaw-go/scripts/benchmark/**`.
- Reduce the Python file count in `bigclaw-go/scripts/benchmark/**` as far as possible without widening scope.
- Record delete/replace/keep rationale for each benchmark Python file touched by this issue.
- Report the impact on the overall repository Python file count.

## Validation

- `cd bigclaw-go && go test ./cmd/bigclawctl`
- `cd bigclaw-go && go test ./cmd/bigclawctl -run 'TestRunAutomationBenchmarkRunMatrixJSONOutput|TestBuildCapacityCertificationReportPassesCheckedInEvidence'`
- `find . -name '*.py' | wc -l`
- `git status --short`

## Results

- Lane-owned Python files under `bigclaw-go/scripts/benchmark/**`:
  - `bigclaw-go/scripts/benchmark/run_matrix.py` -> deleted, replaced by `go run ./cmd/bigclawctl automation benchmark run-matrix`
  - `bigclaw-go/scripts/benchmark/capacity_certification.py` -> deleted, replaced by `go run ./cmd/bigclawctl automation benchmark capacity-certification`
  - `bigclaw-go/scripts/benchmark/capacity_certification_test.py` -> deleted, replaced by Go coverage in `bigclaw-go/cmd/bigclawctl/automation_commands_test.go`
  - `bigclaw-go/scripts/benchmark/soak_local.py` -> deleted, existing Go command `go run ./cmd/bigclawctl automation benchmark soak-local` is now the direct entrypoint
- Remaining benchmark script assets:
  - `bigclaw-go/scripts/benchmark/run_suite.sh` kept because it is already shell-owned and only wraps `go test -bench`
- Repo Python file count impact:
  - before: `123`
  - after: `119`
  - delta: `-4`
