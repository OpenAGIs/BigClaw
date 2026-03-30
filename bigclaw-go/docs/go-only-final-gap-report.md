# Go-Only Final Gap Report

Issue: `BIG-GO-1010`

## Scope Verification

- `bigclaw-go/scripts/e2e/**` still contained Python residue at the start of this sweep.
- `migration/**` has no remaining Python files in this checkout.

Targeted batch at sweep start:

- `bigclaw-go/scripts/e2e/broker_failover_stub_matrix.py`
- `bigclaw-go/scripts/e2e/broker_failover_stub_matrix_test.py`
- `bigclaw-go/scripts/e2e/cross_process_coordination_surface.py`
- `bigclaw-go/scripts/e2e/export_validation_bundle.py`
- `bigclaw-go/scripts/e2e/export_validation_bundle_test.py`
- `bigclaw-go/scripts/e2e/external_store_validation.py`
- `bigclaw-go/scripts/e2e/mixed_workload_matrix.py`
- `bigclaw-go/scripts/e2e/multi_node_shared_queue.py`
- `bigclaw-go/scripts/e2e/multi_node_shared_queue_test.py`
- `bigclaw-go/scripts/e2e/run_all_test.py`
- `bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.py`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py`

## Result

Deleted Python files in this sweep:

- `bigclaw-go/scripts/e2e/broker_failover_stub_matrix_test.py`
- `bigclaw-go/scripts/e2e/export_validation_bundle_test.py`
- `bigclaw-go/scripts/e2e/multi_node_shared_queue_test.py`
- `bigclaw-go/scripts/e2e/run_all_test.py`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.py`

Added Go-native replacement coverage:

- `bigclaw-go/scripts/e2e/run_all_test.go`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.go`

## File Disposition

Deleted or replaced:

- `bigclaw-go/scripts/e2e/broker_failover_stub_matrix_test.py`
  - Deleted.
  - Reason: redundant Python-only unit test. The checked-in broker durability artifacts are already pinned by Go regression coverage under `bigclaw-go/internal/regression/**`.
- `bigclaw-go/scripts/e2e/export_validation_bundle_test.py`
  - Deleted.
  - Reason: redundant Python-only unit test. Generated bundle/index/report surfaces are already pinned by Go regression tests covering live validation summary, index, and follow-up docs.
- `bigclaw-go/scripts/e2e/multi_node_shared_queue_test.py`
  - Deleted.
  - Reason: redundant Python-only unit test. Canonical shared-queue and live takeover proof outputs are already pinned by Go regression tests.
- `bigclaw-go/scripts/e2e/run_all_test.py`
  - Replaced, then deleted.
  - Reason: shell orchestration still matters, but the coverage no longer needs to live in Python. Replaced by `bigclaw-go/scripts/e2e/run_all_test.go`.
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.py`
  - Replaced, then deleted.
  - Reason: policy gate branching still matters, but the coverage no longer needs to live in Python. Replaced by `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.go`.

Kept:

- `bigclaw-go/scripts/e2e/broker_failover_stub_matrix.py`
  - Kept.
  - Reason: no Go-native broker failover evidence generator exists yet.
- `bigclaw-go/scripts/e2e/cross_process_coordination_surface.py`
  - Kept.
  - Reason: no Go-native cross-process capability surface generator exists yet.
- `bigclaw-go/scripts/e2e/export_validation_bundle.py`
  - Kept.
  - Reason: `run_all.sh` still depends on this Python bundle/index exporter.
- `bigclaw-go/scripts/e2e/external_store_validation.py`
  - Kept.
  - Reason: no Go-native external-store validation report generator exists yet.
- `bigclaw-go/scripts/e2e/mixed_workload_matrix.py`
  - Kept.
  - Reason: no Go-native mixed-workload matrix generator exists yet.
- `bigclaw-go/scripts/e2e/multi_node_shared_queue.py`
  - Kept.
  - Reason: no Go-native live multi-node shared-queue proof harness exists yet.
- `bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py`
  - Kept.
  - Reason: no Go-native deterministic subscriber-takeover harness exists yet.
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py`
  - Kept.
  - Reason: no Go-native continuation policy gate command exists yet.
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py`
  - Kept.
  - Reason: no Go-native continuation scorecard generator exists yet.

## Count Impact

- Repository Python files before: `108`
- Repository Python files after: `103`
- Targeted batch Python files before: `14`
- Targeted batch Python files after: `9`
- Net reduction in this sweep: `5`

## Remaining Go-Only Gaps

The repository is not yet Go-only for e2e evidence generation. The remaining blockers are the nine
Python report generators still living under `bigclaw-go/scripts/e2e/**`. They need Go-native
replacements before the e2e/reporting lane can drop Python runtime dependence entirely.
