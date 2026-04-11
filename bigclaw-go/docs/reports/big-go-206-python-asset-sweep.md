# BIG-GO-206 Python Asset Sweep

## Scope

`BIG-GO-206` covers the residual support-asset surfaces that still represent
examples, fixture bundles, demo artifacts, and helper entrypoints after the
Go-only migration work. In this checkout, that support surface is
`bigclaw-go/examples`, `bigclaw-go/docs/reports/broker-failover-stub-artifacts`,
`bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts`, and
`bigclaw-go/scripts/e2e`.

This branch already has no physical `.py` files left to delete, so the lane
lands as regression prevention and evidence capture around the surviving
non-Python support assets.

## Python Baseline

Repository-wide Python file count: `0`.

Audited support-asset directory state:

- `bigclaw-go/examples`: `0` Python files
- `bigclaw-go/docs/reports/broker-failover-stub-artifacts`: `0` Python files
- `bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts`: `0` Python files
- `bigclaw-go/scripts/e2e`: `0` Python files

Explicit remaining Python asset list: none.

## Retired Python Support Helpers

The retired Python E2E support helpers that must remain absent are:

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

## Retained Support Assets

The support surface retained by this lane is now fully non-Python:

- `bigclaw-go/examples/shadow-task.json`
- `bigclaw-go/examples/shadow-corpus-manifest.json`
- `bigclaw-go/docs/reports/broker-failover-stub-artifacts/BF-01/backend-health.json`
- `bigclaw-go/docs/reports/broker-failover-stub-artifacts/BF-01/replay-capture.json`
- `bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts/contention-then-takeover-live/node-a-audit.jsonl`
- `bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts/idle-primary-takeover-live/node-b-audit.jsonl`
- `bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts/lease-expiry-stale-writer-rejected-live/node-a-audit.jsonl`
- `bigclaw-go/scripts/e2e/broker_bootstrap_summary.go`
- `bigclaw-go/scripts/e2e/kubernetes_smoke.sh`
- `bigclaw-go/scripts/e2e/ray_smoke.sh`
- `bigclaw-go/scripts/e2e/run_all.sh`

## Why This Sweep Is Safe

The audited paths still contain JSON examples, fixture bundles, audit-log demo
artifacts, and shell/native helper entrypoints, but those assets are all
repo-native formats rather than executable Python. This lane therefore hardens
the current Go-only support surface instead of claiming fresh Python-file
deletions.

## Validation Commands And Results

- `find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`
  Result: no output; repository-wide Python file count remained `0`.
- `find bigclaw-go/examples bigclaw-go/docs/reports/broker-failover-stub-artifacts bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts bigclaw-go/scripts/e2e -type f -name '*.py' 2>/dev/null | sort`
  Result: no output; the residual support-asset directories stayed Python-free.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO206(RepositoryHasNoPythonFiles|SupportAssetDirectoriesStayPythonFree|RetiredPythonSupportHelpersRemainAbsent|RetainedSupportAssetsRemainAvailable|LaneReportCapturesSweepState)$'`
  Result: `ok  	bigclaw-go/internal/regression	0.149s`
