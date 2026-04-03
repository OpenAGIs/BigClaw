# BIG-GO-1157 Validation

Date: 2026-04-04

## Scope

Issue: `BIG-GO-1157`

Title: `physical Python residual sweep 7`

This lane audited the candidate residual Python assets listed for the sweep and
verified the materialized workspace already contains no live `.py` files. The
targeted benchmark, e2e, migration, and root `scripts/` candidate paths are
already retired in this checkout, while Go and shell entrypoints remain as the
active replacement surface.

Because the workspace baseline is already zero physical Python files, this lane
cannot produce a further numeric drop in `find . -name '*.py' | wc -l`. The
scoped deliverable is therefore validation evidence, explicit blocker capture,
and confirmation that the Go replacement/compatibility paths are intact.

## Delivered

- refreshed `.symphony/workpad.md` for `BIG-GO-1157` with plan, acceptance,
  validation commands, exact results, and the zero-baseline blocker
- confirmed `bigclaw-go/scripts/` only contains active Go and shell surfaces:
  - `bigclaw-go/scripts/benchmark/run_suite.sh`
  - `bigclaw-go/scripts/e2e/broker_bootstrap_summary.go`
  - `bigclaw-go/scripts/e2e/kubernetes_smoke.sh`
  - `bigclaw-go/scripts/e2e/ray_smoke.sh`
  - `bigclaw-go/scripts/e2e/run_all.sh`
- confirmed top-level `scripts/` only contains active shell entrypoints:
  - `scripts/dev_bootstrap.sh`
  - `scripts/ops/bigclaw-issue`
  - `scripts/ops/bigclaw-panel`
  - `scripts/ops/bigclaw-symphony`
  - `scripts/ops/bigclawctl`
- revalidated the regression guards that pin removed candidate Python files to
  absent-on-disk:
  - `bigclaw-go/internal/regression/e2e_entrypoint_migration_test.go`
  - `bigclaw-go/internal/regression/top_level_module_purge_tranche16_test.go`

## Validation

### Python file count

Command:

```bash
find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1157 -name '*.py' | wc -l
```

Result:

```text
0
```

### Replacement surface inventory

Command:

```bash
find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1157/bigclaw-go/scripts -type f | sort
```

Result:

```text
bigclaw-go/scripts/benchmark/run_suite.sh
bigclaw-go/scripts/e2e/broker_bootstrap_summary.go
bigclaw-go/scripts/e2e/kubernetes_smoke.sh
bigclaw-go/scripts/e2e/ray_smoke.sh
bigclaw-go/scripts/e2e/run_all.sh
```

Command:

```bash
find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1157/scripts -type f | sort
```

Result:

```text
scripts/dev_bootstrap.sh
scripts/ops/bigclaw-issue
scripts/ops/bigclaw-panel
scripts/ops/bigclaw-symphony
scripts/ops/bigclawctl
```

### Regression reference scan

Command:

```bash
rg -n "run_task_smoke\.py|export_validation_bundle\.py|validation_bundle_continuation_policy_gate\.py|validation_bundle_continuation_scorecard\.py|broker_failover_stub_matrix\.py|mixed_workload_matrix\.py|cross_process_coordination_surface\.py|subscriber_takeover_fault_matrix\.py|external_store_validation\.py|multi_node_shared_queue\.py" bigclaw-go/internal/regression bigclaw-go/docs
```

Result:

```text
Only matches in bigclaw-go/internal/regression/e2e_entrypoint_migration_test.go
disallowed-path assertions and bigclaw-go/internal/regression/python_test_tranche17_removal_test.go.
```

Command:

```bash
rg -n "scripts/create_issues\.py|scripts/dev_smoke\.py" bigclaw-go/internal/regression
```

Result:

```text
Only matches in bigclaw-go/internal/regression/top_level_module_purge_tranche16_test.go.
```

### Targeted Go regression

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1157/bigclaw-go && go test ./internal/regression -run 'TestE2EScriptDirectoryStaysPythonFree|TestE2EMigrationDocListsOnlyActiveEntrypoints|TestTopLevelModulePurgeTranche16'
```

Result:

```text
ok  	bigclaw-go/internal/regression	0.152s
```

### Go compatibility path

Command:

```bash
bash /Users/openagi/code/bigclaw-workspaces/BIG-GO-1157/scripts/ops/bigclawctl automation e2e external-store-validation --help | head -n 1
```

Result:

```text
usage: bigclawctl automation e2e external-store-validation [flags]
```

## Blocker

The issue acceptance asks for the repo-wide Python file count to decrease, but
this workspace already starts at `0` live `.py` files. There is no remaining
physical Python asset in the materialized checkout for this lane to delete, so a
further numeric reduction is impossible here without changing a different
workspace baseline.
