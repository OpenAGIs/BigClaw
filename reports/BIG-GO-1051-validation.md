# BIG-GO-1051 Validation

Date: 2026-04-01

## Scope

Issue: `BIG-GO-1051`

Title: `Go-replacement U: remove bigclaw-go benchmark Python helpers`

This lane finalizes the benchmark entrypoint cutover under `bigclaw-go/scripts/benchmark/`
by keeping the directory Go-only, removing stale operator references to deleted Python
helpers, and adding a regression check that prevents benchmark `.py` files from
reappearing.

## Delivered

- operator-facing benchmark guidance in `README.md` now points to:
  - `go run ./bigclaw-go/cmd/bigclawctl automation benchmark soak-local ...`
  - `go run ./bigclaw-go/cmd/bigclawctl automation benchmark run-matrix ...`
  - `go run ./bigclaw-go/cmd/bigclawctl automation benchmark capacity-certification ...`
  - `bigclaw-go/scripts/benchmark/run_suite.sh`
- benchmark migration docs now treat the removed Python scripts as retired historical
  paths rather than active entrypoints:
  - `docs/go-cli-script-migration-plan.md`
  - `bigclaw-go/docs/go-cli-script-migration.md`
- `bigclaw-go/cmd/bigclawctl/automation_commands_test.go` now includes
  `TestBenchmarkScriptsStayGoOnly`, which:
  - fails if `bigclaw-go/scripts/benchmark/` contains any `.py` file
  - asserts `run_suite.sh` still routes through `go test -bench ...` and
    `bigclawctl automation benchmark run-matrix`
- rerunning `bigclaw-go/scripts/benchmark/run_suite.sh` refreshed the checked-in benchmark
  evidence:
  - `bigclaw-go/docs/reports/benchmark-report.md`
  - `bigclaw-go/docs/reports/benchmark-matrix-report.json`
  - `bigclaw-go/docs/reports/soak-local-50x8.json`
  - `bigclaw-go/docs/reports/soak-local-100x12.json`

## Validation

### Python file counts

Command:

```bash
find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1051/bigclaw-go/scripts/benchmark -name '*.py' | wc -l
```

Result:

```text
0
```

Command:

```bash
find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1051 -name '*.py' | wc -l
```

Result:

```text
46
```

Note: the benchmark directory was already at `0` Python files in this checkout before the
lane changes landed, so this issue enforces the Go-only state and removes stale entrypoint
references rather than performing a fresh in-branch benchmark `.py` deletion.

### Targeted Go tests

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1051/bigclaw-go && go test ./cmd/bigclawctl/...
```

Result:

```text
ok  	bigclaw-go/cmd/bigclawctl	4.697s
```

### Benchmark command help checks

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1051/bigclaw-go && go run ./cmd/bigclawctl automation benchmark soak-local --help
```

Result: exit code `0`, printed `usage: bigclawctl automation benchmark soak-local [flags]`

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1051/bigclaw-go && go run ./cmd/bigclawctl automation benchmark run-matrix --help
```

Result: exit code `0`, printed `usage: bigclawctl automation benchmark run-matrix [flags]`

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1051/bigclaw-go && go run ./cmd/bigclawctl automation benchmark capacity-certification --help
```

Result: exit code `0`, printed `usage: bigclawctl automation benchmark capacity-certification [flags]`

### Benchmark wrapper execution

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1051/bigclaw-go && ./scripts/benchmark/run_suite.sh
```

Result:

- exit code `0`
- regenerated `docs/reports/benchmark-report.md`
- regenerated `docs/reports/benchmark-matrix-report.json`
- regenerated `docs/reports/soak-local-50x8.json`
- regenerated `docs/reports/soak-local-100x12.json`

Observed benchmark output included:

```text
BenchmarkMemoryQueueEnqueueLease-10    	   40660	     29227 ns/op
BenchmarkFileQueueEnqueueLease-10      	      24	  48228771 ns/op
BenchmarkSQLiteQueueEnqueueLease-10    	     128	   9340130 ns/op
BenchmarkSchedulerDecide-10    	 2735659	       436.0 ns/op
```

## Commit And Push

- Commit: `9746a50c`
- Message: `BIG-GO-1051: finalize Go-only benchmark entrypoints`
- Push: `git push origin main` succeeded

## Residual Risk

- This lane does not reduce the repo-wide `.py` count because the benchmark Python helpers had
  already been deleted before the branch was created.
- The remaining Go-only enforcement for benchmark entrypoints now depends on the new regression
  test and on keeping docs aligned with the active `bigclawctl automation benchmark ...`
  commands.
