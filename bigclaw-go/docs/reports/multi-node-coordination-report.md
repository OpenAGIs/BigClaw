# Multi-Node Coordination Report

## Scope

- Run date: 2026-03-15
- Command: `python3 scripts/e2e/multi_node_shared_queue.py --count 200 --submit-workers 8 --timeout-seconds 180 --state-dir tmp/shared-queue-proof --bundle-dir docs/reports/shared-queue-takeover-evidence/latest --report-path docs/reports/multi-node-shared-queue-report.json`
- Stable bundle: `docs/reports/shared-queue-takeover-evidence/latest`
- Goal: produce a concrete two-node shared-queue proof for the current Go control plane.

## Result

- Total tasks: `200`
- Submitted by `node-a`: `100`
- Submitted by `node-b`: `100`
- Completed by `node-a`: `99`
- Completed by `node-b`: `101`
- Cross-node completions: `113`
- Duplicate `task.started`: `0`
- Duplicate `task.completed`: `0`
- Missing terminal completions: `0`

## Meaning

This run proves that two independent `bigclawd` processes can share the same SQLite-backed queue and coordinate task consumption without duplicate terminal execution in the current local topology. It is not a full leader-election system, but it gives the epic a concrete multi-node coordination proof instead of relying only on single-process evidence.

## Artifact

- `docs/reports/multi-node-shared-queue-report.json`
- `docs/reports/shared-queue-takeover-evidence/latest`
