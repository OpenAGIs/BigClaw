# BIG-GO-930 Go-only Repo Cutover

## Scope

This slice closes the already-migrated Python entrypoints that only proxied into
`bigclaw-go/cmd/bigclawctl`, documents the remaining non-Go inventory, and defines
the merge bar for a Go-first repository posture.

## Removed in this slice

- `bigclaw-go/scripts/e2e/run_task_smoke.py`
- `bigclaw-go/scripts/benchmark/soak_local.py`
- `bigclaw-go/scripts/migration/shadow_compare.py`

All three paths were pure Python shims over:

- `go run ./cmd/bigclawctl automation e2e run-task-smoke`
- `go run ./cmd/bigclawctl automation benchmark soak-local`
- `go run ./cmd/bigclawctl automation migration shadow-compare`

The active callers now invoke the Go CLI directly from:

- `bigclaw-go/scripts/e2e/run_all.sh`
- `bigclaw-go/scripts/e2e/kubernetes_smoke.sh`
- `bigclaw-go/scripts/e2e/ray_smoke.sh`
- `bigclaw-go/scripts/benchmark/run_matrix.py`

## Current Python / non-Go inventory

### Frozen legacy Python runtime surface

- `src/bigclaw/*.py`
- `setup.py`
- `pyproject.toml`
- `.pre-commit-config.yaml`

Delete condition:
- every remaining migration-only runtime surface has a verified Go owner under
  `bigclaw-go/internal/*` or `bigclaw-go/cmd/*`
- the root `tests/*.py` suite is either removed or rewritten against Go-owned
  fixtures and CLI/report surfaces
- `bigclawctl legacy-python compile-check --json` is no longer required by any
  release or reviewer workflow

### Legacy Python tests

- `tests/conftest.py`
- `tests/*.py`

Delete condition:
- coverage is replaced by `go test` in `bigclaw-go/...`
- any remaining repo-root Python helper modules are gone

### Remaining `bigclaw-go` Python automation with real logic

- `bigclaw-go/scripts/e2e/export_validation_bundle.py`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py`
- `bigclaw-go/scripts/e2e/multi_node_shared_queue.py`
- `bigclaw-go/scripts/e2e/mixed_workload_matrix.py`
- `bigclaw-go/scripts/e2e/external_store_validation.py`
- `bigclaw-go/scripts/e2e/cross_process_coordination_surface.py`
- `bigclaw-go/scripts/e2e/broker_failover_stub_matrix.py`
- `bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py`
- `bigclaw-go/scripts/benchmark/capacity_certification.py`
- `bigclaw-go/scripts/benchmark/run_matrix.py`
- `bigclaw-go/scripts/migration/export_live_shadow_bundle.py`
- `bigclaw-go/scripts/migration/live_shadow_scorecard.py`
- `bigclaw-go/scripts/migration/shadow_matrix.py`

Go replacement / migration path:
- move each report generator into `bigclaw-go/cmd/bigclawctl automation ...` or a
  dedicated Go reporting package under `bigclaw-go/internal/*`
- keep JSON payload shape stable by pinning the existing checked-in reports with
  regression tests before removing the Python file

Delete condition:
- a Go command exists
- README/docs point to the Go command, not the Python file
- the repo regression tests assert the canonical report shape
- one rollout cycle completes without new operator references to the Python path

### Intentional non-Go wrappers that remain active

- `scripts/dev_bootstrap.sh`
- `scripts/ops/bigclawctl`
- `scripts/ops/bigclaw-issue`
- `scripts/ops/bigclaw-panel`
- `scripts/ops/bigclaw-symphony`

Rationale:
- these are operator wrappers, not implementation ownership
- `scripts/ops/bigclawctl` still standardizes `--repo` resolution and `go run`
  invocation from the repo root

Delete condition:
- ship a compiled `bigclawctl` distribution path or another stable operator entrypoint

## Main merge strategy

Merge `main` only when all of the following remain true:

1. `bigclaw-go/scripts/e2e/run_all.sh`, `kubernetes_smoke.sh`, `ray_smoke.sh`, and
   `scripts/benchmark/run_matrix.py` call `go run ./cmd/bigclawctl ...` directly.
2. The deleted shim paths above do not exist.
3. No README or active workflow doc presents the deleted Python shims as a supported path.
4. The remaining Python inventory is explicitly limited to the categories in this document.
5. `go test ./...` from `bigclaw-go/` stays green.

## Regression validation commands

```bash
cd bigclaw-go && go test ./...
cd bigclaw-go && python3 scripts/e2e/run_all_test.py
cd bigclaw-go && go run ./cmd/bigclawctl automation e2e run-task-smoke --help
cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark soak-local --help
cd bigclaw-go && go run ./cmd/bigclawctl automation migration shadow-compare --help
cd .. && rg --files -g '*.py' -g '*.sh'
cd .. && test ! -e bigclaw-go/scripts/e2e/run_task_smoke.py
cd .. && test ! -e bigclaw-go/scripts/benchmark/soak_local.py
cd .. && test ! -e bigclaw-go/scripts/migration/shadow_compare.py
```
