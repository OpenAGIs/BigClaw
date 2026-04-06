# BIG-GO-1553 Python Asset Sweep

## Scope

Refill lane `BIG-GO-1553` records the exact historical deletion ledger for the
remaining `bigclaw-go/scripts/**/*.py` surface and verifies that the current
checkout keeps that surface at zero physical Python files on disk.

## Historical Baseline And Current Counts

- Historical `bigclaw-go/scripts` physical Python file count at
  `fdb20c43` (`8ebdd50d^`): `23`
- Current `bigclaw-go/scripts` physical Python file count on disk: `0`
- Exact `bigclaw-go/scripts` count delta: `-23`
- Current repository-wide physical Python file count on disk: `0`

The deletions that produced the current zero-file state already landed before
this refill lane started, so the lane ships exact historical evidence,
validation output, and a regression guard that keeps the `bigclaw-go/scripts`
surface Python-free.

## Exact Deleted-File Ledger

Historical deleted files for `bigclaw-go/scripts` between `fdb20c43`
(`8ebdd50d^`) and `HEAD`:

- `8ebdd50d`: `bigclaw-go/scripts/e2e/run_task_smoke.py`
- `8ebdd50d`: `bigclaw-go/scripts/migration/shadow_compare.py`
- `5afb870f`: `bigclaw-go/scripts/migration/shadow_matrix.py`
- `546c0c64`: `bigclaw-go/scripts/migration/live_shadow_scorecard.py`
- `68221e4e`: `bigclaw-go/scripts/migration/export_live_shadow_bundle.py`
- `da168148`: `bigclaw-go/scripts/benchmark/capacity_certification.py`
- `da168148`: `bigclaw-go/scripts/benchmark/capacity_certification_test.py`
- `da168148`: `bigclaw-go/scripts/benchmark/run_matrix.py`
- `da168148`: `bigclaw-go/scripts/benchmark/soak_local.py`
- `2ed49341`: `bigclaw-go/scripts/e2e/export_validation_bundle.py`
- `2ed49341`: `bigclaw-go/scripts/e2e/export_validation_bundle_test.py`
- `2ed49341`: `bigclaw-go/scripts/e2e/run_all_test.py`
- `2ed49341`: `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py`
- `2ed49341`: `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.py`
- `2ed49341`: `bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py`
- `ed633b9d`: `bigclaw-go/scripts/e2e/broker_failover_stub_matrix.py`
- `ed633b9d`: `bigclaw-go/scripts/e2e/broker_failover_stub_matrix_test.py`
- `32033874`: `bigclaw-go/scripts/e2e/mixed_workload_matrix.py`
- `1f6e9876`: `bigclaw-go/scripts/e2e/cross_process_coordination_surface.py`
- `3f32b502`: `bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py`
- `c225a50c`: `bigclaw-go/scripts/e2e/multi_node_shared_queue_test.py`
- `ad32285c`: `bigclaw-go/scripts/e2e/external_store_validation.py`
- `42363805`: `bigclaw-go/scripts/e2e/multi_node_shared_queue.py`

## Replacement Paths

The active replacement surface for the retired Python scripts includes:

- `bigclaw-go/scripts/benchmark/run_suite.sh`
- `bigclaw-go/scripts/e2e/run_all.sh`
- `bigclaw-go/scripts/e2e/kubernetes_smoke.sh`
- `bigclaw-go/scripts/e2e/ray_smoke.sh`
- `bigclaw-go/scripts/e2e/broker_bootstrap_summary.go`
- `bigclaw-go/cmd/bigclawctl/automation_benchmark_commands.go`
- `bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands.go`
- `bigclaw-go/cmd/bigclawctl/automation_e2e_coordination_surface_command.go`
- `bigclaw-go/cmd/bigclawctl/automation_e2e_external_store_validation_command.go`
- `bigclaw-go/cmd/bigclawctl/automation_e2e_mixed_workload_command.go`
- `bigclaw-go/cmd/bigclawctl/automation_e2e_multi_node_shared_queue_command.go`
- `bigclaw-go/cmd/bigclawctl/automation_e2e_takeover_matrix_command.go`

## Validation Commands And Results

- `find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`
  Result: no output; repository-wide Python file count remained `0`.
- `find bigclaw-go/scripts -type f -name '*.py' | sort`
  Result: no output; `bigclaw-go/scripts` remained at `0` physical Python
  files.
- `git ls-tree -r --name-only fdb20c43 bigclaw-go/scripts | rg '\.py$'`
  Result: `23` historical `bigclaw-go/scripts` Python paths at the pre-deletion
  baseline.
- `git log --diff-filter=D --summary -- bigclaw-go/scripts`
  Result: the exact deleted-file ledger above, sourced from repository history.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1553(RepositoryHasNoPythonFiles|BigclawGoScriptsStayPythonFree|ReplacementPathsRemainAvailable|LaneReportCapturesExactDeltaAndLedger)$'`
  Result: `ok  	bigclaw-go/internal/regression	3.168s`
