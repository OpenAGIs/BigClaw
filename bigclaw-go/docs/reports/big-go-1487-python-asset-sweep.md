# BIG-GO-1487 Python Asset Sweep

## Scope

Largest-directory-first sweep on branch `BIG-GO-1480` residual Python assets.

- Before total Python file count: `23`
- Before directory counts:
  - `bigclaw-go/scripts/e2e`: `15`
  - `bigclaw-go/scripts/migration`: `4`
  - `bigclaw-go/scripts/benchmark`: `4`
- Swept directory: `bigclaw-go/scripts/e2e`
- After total Python file count: `8`
- After directory counts:
  - `bigclaw-go/scripts/migration`: `4`
  - `bigclaw-go/scripts/benchmark`: `4`

## Deleted Files

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

## Validation

- `bash -n bigclaw-go/scripts/e2e/run_all.sh && bash -n bigclaw-go/scripts/e2e/kubernetes_smoke.sh && bash -n bigclaw-go/scripts/e2e/ray_smoke.sh`
  - Result: passed
- `go test ./cmd/bigclawctl -run 'TestRunAutomationRunTaskSmokeJSONOutput|TestAutomationExternalStoreValidationWritesReport|TestAutomationMixedWorkloadMatrixBuildsReport|TestAutomationCrossProcessCoordinationSurfaceBuildsReport|TestAutomationMultiNodeSharedQueueBuildLiveTakeoverReport|TestAutomationSubscriberTakeoverFaultMatrixBuildsReport|TestAutomationContinuationPolicyGateReturnsPolicyGoWhenInputsPass|TestRunAllUsesGoBundleCommandsAndDefaultsHoldMode'`
  - Result: passed
- `go test ./internal/regression -run 'TestE2EScriptDirectoryStaysPythonFree|TestE2EMigrationDocListsOnlyActiveEntrypoints|TestProviderLiveHandoffSurfaceStaysAligned'`
  - Result: passed
