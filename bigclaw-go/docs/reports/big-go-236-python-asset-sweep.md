# BIG-GO-236 Python Asset Sweep

## Scope

`BIG-GO-236` covers the residual support-asset surfaces that still represent
examples, fixture bundles, demo artifacts, and helper entrypoints after the
Go-only migration work. In this checkout, that support surface is
`bigclaw-go/examples`, `bigclaw-go/docs/reports/broker-failover-stub-artifacts`,
`bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts`,
`bigclaw-go/docs/reports/live-shadow-runs`,
`bigclaw-go/docs/reports/live-validation-runs`, and `scripts/ops`.

This branch already has no physical `.py` files left to delete, so the lane
lands as regression prevention and evidence capture around the surviving
non-Python support assets.

## Python Baseline

Repository-wide Python file count: `0`.

Audited support-asset directory state:

- `bigclaw-go/examples`: `0` Python files
- `bigclaw-go/docs/reports/broker-failover-stub-artifacts`: `0` Python files
- `bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts`: `0` Python files
- `bigclaw-go/docs/reports/live-shadow-runs`: `0` Python files
- `bigclaw-go/docs/reports/live-validation-runs`: `0` Python files
- `scripts/ops`: `0` Python files

Explicit remaining Python asset list: none.

## Retained Support Assets

The support surface retained by this lane is now fully non-Python:

- `bigclaw-go/examples/shadow-task.json`
- `bigclaw-go/examples/shadow-task-budget.json`
- `bigclaw-go/examples/shadow-task-validation.json`
- `bigclaw-go/examples/shadow-corpus-manifest.json`
- `bigclaw-go/docs/reports/broker-failover-stub-artifacts/BF-08/replay-capture.json`
- `bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts/contention-then-takeover-live/node-a-audit.jsonl`
- `bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/shadow-matrix-report.json`
- `bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/shared-queue-companion-summary.json`
- `scripts/ops/bigclawctl`
- `scripts/ops/bigclaw-issue`
- `scripts/ops/bigclaw-panel`
- `scripts/ops/bigclaw-symphony`

## Why This Sweep Is Safe

The audited paths still contain JSON manifests, fixture bundles, audit logs,
checked-in validation outputs, and shell helper entrypoints, but those assets
are all native repo formats rather than executable Python. This lane therefore
hardens the current Go-only support surface instead of claiming fresh
Python-file deletions.

## Validation Commands And Results

- `find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`
  Result: no output; repository-wide Python file count remained `0`.
- `find bigclaw-go/examples bigclaw-go/docs/reports/broker-failover-stub-artifacts bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts bigclaw-go/docs/reports/live-shadow-runs bigclaw-go/docs/reports/live-validation-runs scripts/ops -type f -name '*.py' 2>/dev/null | sort`
  Result: no output; the residual support-asset directories stayed Python-free.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO236(RepositoryHasNoPythonFiles|SupportAssetDirectoriesStayPythonFree|RetainedSupportAssetsRemainAvailable|LaneReportCapturesSweepState)$'`
  Result: `ok  	bigclaw-go/internal/regression	0.149s`
