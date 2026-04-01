# BIG-GO-1067 Validation

Date: 2026-04-01

## Scope

Issue: `BIG-GO-1067`

Title: `bigclaw-go scripts e2e residual sweep A`

This umbrella lane covers the residual Python asset batch called out for
`bigclaw-go/scripts/benchmark/` and `bigclaw-go/scripts/e2e/`. The underlying code
migration is already present on `main` through the earlier benchmark closeout
(`BIG-GO-1051`) and e2e closeout (`BIG-GO-1053`). This report records the missing
issue-scoped evidence for the combined sweep and re-validates the active Go-only
replacement paths.

## Asset List

### Benchmark tranche

- `bigclaw-go/scripts/benchmark/capacity_certification.py`
- `bigclaw-go/scripts/benchmark/capacity_certification_test.py`
- `bigclaw-go/scripts/benchmark/run_matrix.py`
- `bigclaw-go/scripts/benchmark/soak_local.py`

Active replacements:

- `go run ./bigclaw-go/cmd/bigclawctl automation benchmark soak-local ...`
- `go run ./bigclaw-go/cmd/bigclawctl automation benchmark run-matrix ...`
- `go run ./bigclaw-go/cmd/bigclawctl automation benchmark capacity-certification ...`
- `bigclaw-go/scripts/benchmark/run_suite.sh`

### E2E tranche

- `bigclaw-go/scripts/e2e/broker_failover_stub_matrix.py`
- `bigclaw-go/scripts/e2e/broker_failover_stub_matrix_test.py`
- `bigclaw-go/scripts/e2e/cross_process_coordination_surface.py`
- `bigclaw-go/scripts/e2e/export_validation_bundle.py`
- `bigclaw-go/scripts/e2e/export_validation_bundle_test.py`
- `bigclaw-go/scripts/e2e/external_store_validation.py`
- `bigclaw-go/scripts/e2e/mixed_workload_matrix.py`
- `bigclaw-go/scripts/e2e/multi_node_shared_queue.py`

Active replacements:

- `go run ./bigclaw-go/cmd/bigclawctl automation e2e broker-failover-stub-matrix ...`
- `go run ./bigclaw-go/cmd/bigclawctl automation e2e mixed-workload-matrix ...`
- `go run ./bigclaw-go/cmd/bigclawctl automation e2e cross-process-coordination-surface ...`
- `go run ./bigclaw-go/cmd/bigclawctl automation e2e export-validation-bundle ...`
- `go run ./bigclaw-go/cmd/bigclawctl automation e2e external-store-validation ...`
- `go run ./bigclaw-go/cmd/bigclawctl automation e2e multi-node-shared-queue ...`
- `bigclaw-go/scripts/e2e/run_all.sh`
- `bigclaw-go/scripts/e2e/kubernetes_smoke.sh`
- `bigclaw-go/scripts/e2e/ray_smoke.sh`

## Delivered

- Confirmed `bigclaw-go/scripts/benchmark/` and `bigclaw-go/scripts/e2e/` are both
  Python-free in the current checkout.
- Confirmed benchmark Go replacement coverage remains enforced by
  `bigclaw-go/cmd/bigclawctl/automation_commands_test.go`.
- Confirmed e2e Go replacement coverage remains enforced by
  `bigclaw-go/internal/regression/e2e_entrypoint_migration_test.go`.
- Added issue-scoped workpad and closeout artifacts for the residual sweep so this
  umbrella ticket has its own validation trail rather than relying only on `BIG-GO-1051`
  and `BIG-GO-1053`.

## Validation

### Python file counts

Command:

```bash
find bigclaw-go/scripts/benchmark -maxdepth 1 -name '*.py' | wc -l
```

Result:

```text
0
```

Command:

```bash
find bigclaw-go/scripts/e2e -maxdepth 1 -name '*.py' | wc -l
```

Result:

```text
0
```

Command:

```bash
find . -name '*.py' | wc -l
```

Result:

```text
43
```

### Stale reference scan

Command:

```bash
rg -n "capacity_certification\.py|capacity_certification_test\.py|run_matrix\.py|soak_local\.py|broker_failover_stub_matrix\.py|broker_failover_stub_matrix_test\.py|cross_process_coordination_surface\.py|export_validation_bundle\.py|export_validation_bundle_test\.py|external_store_validation\.py|mixed_workload_matrix\.py|multi_node_shared_queue\.py" tests bigclaw-go docs README.md workflow.md .github . -g '!reports/**' -g '!.symphony/workpad.md' 2>/dev/null
```

Result:

```text
matches are limited to historical tracker notes in local-issues.json and the live regression guard in bigclaw-go/internal/regression/e2e_entrypoint_migration_test.go
```

### Targeted Go tests

Command:

```bash
cd bigclaw-go && go test ./cmd/bigclawctl/... ./internal/regression/...
```

Result:

```text
ok  	bigclaw-go/cmd/bigclawctl	4.237s
ok  	bigclaw-go/internal/regression	1.186s
```

### Benchmark command help checks

Command:

```bash
cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark soak-local --help
```

Result: exit code `0`, printed `usage: bigclawctl automation benchmark soak-local [flags]`

Command:

```bash
cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark run-matrix --help
```

Result: exit code `0`, printed `usage: bigclawctl automation benchmark run-matrix [flags]`

Command:

```bash
cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark capacity-certification --help
```

Result: exit code `0`, printed `usage: bigclawctl automation benchmark capacity-certification [flags]`

Command:

```bash
cd bigclaw-go && ./scripts/benchmark/run_suite.sh
```

Result:

- exit code `0`
- benchmark output included:
  - `BenchmarkMemoryQueueEnqueueLease-10    	   38758	     29831 ns/op`
  - `BenchmarkFileQueueEnqueueLease-10      	      22	  46735343 ns/op`
  - `BenchmarkSQLiteQueueEnqueueLease-10    	     100	  10345601 ns/op`
  - `BenchmarkSchedulerDecide-10    	 2678006	       425.5 ns/op`

### E2E command help checks

Command:

```bash
cd bigclaw-go && go run ./cmd/bigclawctl automation e2e broker-failover-stub-matrix --help
```

Result: exit code `0`, printed `usage: bigclawctl automation e2e broker-failover-stub-matrix [flags]`

Command:

```bash
cd bigclaw-go && go run ./cmd/bigclawctl automation e2e mixed-workload-matrix --help
```

Result: exit code `0`, printed `usage: bigclawctl automation e2e mixed-workload-matrix [flags]`

Command:

```bash
cd bigclaw-go && go run ./cmd/bigclawctl automation e2e cross-process-coordination-surface --help
```

Result: exit code `0`, printed `usage: bigclawctl automation e2e cross-process-coordination-surface [flags]`

Command:

```bash
cd bigclaw-go && go run ./cmd/bigclawctl automation e2e export-validation-bundle --help
```

Result: exit code `0`, printed `usage: bigclawctl automation e2e export-validation-bundle [flags]`

Command:

```bash
cd bigclaw-go && go run ./cmd/bigclawctl automation e2e external-store-validation --help
```

Result: exit code `0`, printed `usage: bigclawctl automation e2e external-store-validation [flags]`

Command:

```bash
cd bigclaw-go && go run ./cmd/bigclawctl automation e2e multi-node-shared-queue --help
```

Result: exit code `0`, printed `usage: bigclawctl automation e2e multi-node-shared-queue [flags]`

## Python Count Impact

- Current repo-wide count: `43` tracked `.py` files.
- Earlier benchmark closeout evidence for `BIG-GO-1051` recorded `46` repo-wide `.py`
  files before the later e2e stale-test removals were fully reflected.
- The combined residual sweep therefore lands at least a `-3` repo-wide `.py` delta
  across the already-landed `BIG-GO-1051` and `BIG-GO-1053` evidence window, while the
  full target asset list for this umbrella issue is now absent from the live script
  surface.

## Residual Risk

- This ticket does not introduce a fresh code-path deletion in the current branch because
  the target Python assets were already removed on `main` before the umbrella evidence was
  assembled here.
- Historical references remain in archived reports and `local-issues.json`; those are kept
  as audit trail rather than active entrypoints.
