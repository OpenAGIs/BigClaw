# BIG-GO-17 Auxiliary Nested Python Sweep B

## Scope

`BIG-GO-17` records a residual sweep for nested auxiliary report and evidence
directories that sit outside the primary source and script paths.

## Remaining Python Inventory

Repository-wide Python file count: `0`.

- `docs/reports`: `0` Python files
- `bigclaw-go/docs/reports/broker-failover-stub-artifacts`: `0` Python files
- `bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts`: `0` Python files
- `bigclaw-go/docs/reports/live-shadow-runs`: `0` Python files
- `bigclaw-go/docs/reports/live-validation-runs`: `0` Python files

This checkout therefore lands as a regression-prevention sweep for nested
auxiliary evidence surfaces rather than a direct Python-file deletion batch.

## Retained Native Evidence Assets

- `docs/reports/bootstrap-cache-validation.md`
- `bigclaw-go/docs/reports/broker-failover-stub-artifacts/BF-05/publish-attempt-ledger.json`
- `bigclaw-go/docs/reports/broker-failover-stub-artifacts/BF-07/fault-timeline.json`
- `bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts/contention-then-takeover-live/node-a-audit.jsonl`
- `bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts/lease-expiry-stale-writer-rejected-live/node-b-audit.jsonl`
- `bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/summary.json`
- `bigclaw-go/docs/reports/live-validation-runs/20260314T164647Z/summary.json`
- `bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/broker-validation-summary.json`

## Validation Commands And Results

- `find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`
  Result: no output; repository-wide Python file count remained `0`.
- `find docs/reports bigclaw-go/docs/reports/broker-failover-stub-artifacts bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts bigclaw-go/docs/reports/live-shadow-runs bigclaw-go/docs/reports/live-validation-runs -type f -name '*.py' 2>/dev/null | sort`
  Result: no output; the nested auxiliary report and evidence directories remained Python-free.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO17(NestedAuxiliaryDirectoriesStayPythonFree|RetainedNativeEvidenceAssetsRemainAvailable|LaneReportCapturesSweepState)$'`
  Result: `ok  	bigclaw-go/internal/regression	0.454s`
