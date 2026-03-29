# BIG-GO-970 Validation

## Scope

- `bigclaw-go/scripts/e2e/**`
- `bigclaw-go/scripts/migration/**`

## Python File Inventory And Disposition

| File | Disposition | Reason |
| --- | --- | --- |
| `bigclaw-go/scripts/e2e/broker_failover_stub_matrix.py` | retained | 984-line report generator with checked-in broker failover evidence contract and no Go replacement in this slice. |
| `bigclaw-go/scripts/e2e/broker_failover_stub_matrix_test.py` | deleted | Redundant script-local unit test; checked-in broker failover/report surfaces are already covered by Go regression and durability tests. |
| `bigclaw-go/scripts/e2e/cross_process_coordination_surface.py` | retained | 268-line report generator tied to checked-in capability-surface JSON and docs, with no direct Go CLI replacement yet. |
| `bigclaw-go/scripts/e2e/export_validation_bundle.py` | retained | 803-line bundle exporter still owns live validation summary/index generation; no Go replacement yet. |
| `bigclaw-go/scripts/e2e/export_validation_bundle_test.py` | deleted | Duplicate Python test surface; exporter behavior remains covered by top-level `tests/test_parallel_validation_bundle.py`. |
| `bigclaw-go/scripts/e2e/external_store_validation.py` | retained | 512-line evidence/report generator still referenced by docs and regression surfaces. |
| `bigclaw-go/scripts/e2e/mixed_workload_matrix.py` | retained | 224-line matrix/report generator with no Go replacement in this issue. |
| `bigclaw-go/scripts/e2e/multi_node_shared_queue.py` | retained | 856-line live proof harness used by checked-in shared-queue/takeover evidence and continuation surfaces. |
| `bigclaw-go/scripts/e2e/multi_node_shared_queue_test.py` | deleted | Redundant script-local test; checked-in shared-queue report and companion summary already have Go regression coverage. |
| `bigclaw-go/scripts/e2e/run_all_test.py` | deleted | Script-local orchestration test duplicated by the broader Python validation-bundle harness in `tests/test_parallel_validation_bundle.py`. |
| `bigclaw-go/scripts/e2e/run_task_smoke.py` | replaced then deleted | Already migrated to `go run ./cmd/bigclawctl automation e2e run-task-smoke`; shell wrappers and docs now call the Go command directly. |
| `bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py` | retained | 708-line deterministic takeover harness still backs checked-in takeover evidence and has no Go command replacement yet. |
| `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py` | retained | Active policy-gate generator still consumed by checked-in artifacts and distributed diagnostics surfaces. |
| `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.py` | deleted | Duplicate script-local test; behavior remains covered by top-level `tests/test_validation_bundle_continuation_policy_gate.py`. |
| `bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py` | retained | Active scorecard generator still consumed by continuation artifacts and API surfaces. |
| `bigclaw-go/scripts/migration/export_live_shadow_bundle.py` | retained | Bundle/index generator for live-shadow review pack with no Go replacement in this slice. |
| `bigclaw-go/scripts/migration/live_shadow_scorecard.py` | retained | Scorecard generator for checked-in live-shadow evidence, still referenced by docs and regression tests. |
| `bigclaw-go/scripts/migration/shadow_compare.py` | replaced then deleted | `shadow_matrix.py` now shells out to `go run ./cmd/bigclawctl automation migration shadow-compare`, so the thin Python shim is no longer needed. |
| `bigclaw-go/scripts/migration/shadow_matrix.py` | retained | Still a Python matrix/report generator, but now depends on the Go CLI directly instead of importing a Python shim module. |

## Additional In-Scope Fix

- `bigclaw-go/scripts/e2e/export_validation_bundle.py` now opts into deferred annotation evaluation via `from __future__ import annotations`, so the checked-in exporter remains executable under the repository's current `python3` interpreter while this material-pass cleanup removes adjacent script-local tests.

## Count Impact

- Total repository Python files before: `123`
- Total repository Python files after: `116`
- Net change across repository: `-7`
- Target directory Python files before: `19`
- Target directory Python files after: `12`
- Net change in `bigclaw-go/scripts/e2e/**` and `bigclaw-go/scripts/migration/**`: `-7`

## Validation Commands

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-970/bigclaw-go && go test ./cmd/bigclawctl/... ./internal/regression/...
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-970 && PYTHONPATH=src python3 -m pytest tests/test_parallel_validation_bundle.py tests/test_validation_bundle_continuation_policy_gate.py tests/test_live_shadow_bundle.py -q
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-970 && bash -n bigclaw-go/scripts/e2e/kubernetes_smoke.sh
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-970 && bash -n bigclaw-go/scripts/e2e/ray_smoke.sh
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-970 && bash -n bigclaw-go/scripts/e2e/run_all.sh
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-970 && find . -name '*.py' | wc -l
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-970 && find bigclaw-go/scripts/e2e bigclaw-go/scripts/migration -name '*.py' | wc -l
```

## Validation Results

- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-970/bigclaw-go && go test ./cmd/bigclawctl/... ./internal/regression/...`
  - `ok  	bigclaw-go/cmd/bigclawctl	(cached)`
  - `ok  	bigclaw-go/internal/regression	0.675s`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-970 && PYTHONPATH=src python3 -m pytest tests/test_parallel_validation_bundle.py tests/test_validation_bundle_continuation_policy_gate.py tests/test_live_shadow_bundle.py -q`
  - `8 passed in 0.20s`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-970 && bash -n bigclaw-go/scripts/e2e/kubernetes_smoke.sh`
  - `bash -n kubernetes_smoke.sh: PASS`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-970 && bash -n bigclaw-go/scripts/e2e/ray_smoke.sh`
  - `bash -n ray_smoke.sh: PASS`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-970 && bash -n bigclaw-go/scripts/e2e/run_all.sh`
  - `bash -n run_all.sh: PASS`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-970/bigclaw-go && go run ./cmd/bigclawctl automation e2e run-task-smoke --help`
  - exited `0`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-970/bigclaw-go && go run ./cmd/bigclawctl automation migration shadow-compare --help`
  - exited `0`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-970 && python3 -m py_compile bigclaw-go/scripts/migration/shadow_matrix.py`
  - `py_compile shadow_matrix.py: PASS`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-970 && find . -name '*.py' | wc -l`
  - `116`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-970 && find bigclaw-go/scripts/e2e bigclaw-go/scripts/migration -name '*.py' | wc -l`
  - `12`
