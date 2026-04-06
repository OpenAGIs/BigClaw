# BIG-GO-1545

## Plan

1. Materialize the issue workspace on a valid `BIG-GO-1545` branch from `origin/main`.
2. Capture baseline `.py` counts for the repository and for the remaining `bigclaw-go/scripts/**/*.py` reporting/observability sweep surface.
3. Physically delete the in-scope Python files.
4. Recount after deletion and record the exact removed-file list.
5. Run targeted validation commands, then commit and push the branch.

## Acceptance

- Remaining reporting/observability Python files under `bigclaw-go/scripts` are physically removed.
- Before/after counts are recorded.
- The exact removed-file list is recorded.
- Validation commands and results are recorded.
- Changes remain scoped to this sweep.
- The branch is committed and pushed to `origin/BIG-GO-1545`.

## Validation

- `find . -type f -name '*.py' | sort > /tmp/BIG-GO-1545-all-py-before.txt && wc -l < /tmp/BIG-GO-1545-all-py-before.txt | tr -d ' '` -> `138`
- `find bigclaw-go/scripts -type f -name '*.py' | sort > /tmp/BIG-GO-1545-removed-files-before.txt && wc -l < /tmp/BIG-GO-1545-removed-files-before.txt | tr -d ' '` -> `23`
- `find . -type f -name '*.py' | sort > /tmp/BIG-GO-1545-all-py-after.txt && wc -l < /tmp/BIG-GO-1545-all-py-after.txt | tr -d ' '` -> `115`
- `find bigclaw-go/scripts -type f -name '*.py' | sort > /tmp/BIG-GO-1545-in-scope-after.txt && wc -l < /tmp/BIG-GO-1545-in-scope-after.txt | tr -d ' '` -> `0`
- `git diff --name-only --diff-filter=D | sort` -> 23 deleted files listed below.

## Before/After Counts

- Repository `.py` files: `138 -> 115`
- In-scope `bigclaw-go/scripts/**/*.py` files: `23 -> 0`

## Removed Files

- `bigclaw-go/scripts/benchmark/capacity_certification.py`
- `bigclaw-go/scripts/benchmark/capacity_certification_test.py`
- `bigclaw-go/scripts/benchmark/run_matrix.py`
- `bigclaw-go/scripts/benchmark/soak_local.py`
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
- `bigclaw-go/scripts/e2e/run_task_smoke.py`
- `bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.py`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py`
- `bigclaw-go/scripts/migration/export_live_shadow_bundle.py`
- `bigclaw-go/scripts/migration/live_shadow_scorecard.py`
- `bigclaw-go/scripts/migration/shadow_compare.py`
- `bigclaw-go/scripts/migration/shadow_matrix.py`
