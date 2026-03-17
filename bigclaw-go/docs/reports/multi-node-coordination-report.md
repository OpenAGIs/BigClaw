# Multi-Node Coordination Report

## Scope

- Run date: 2026-03-13
- Command: `python3 scripts/e2e/multi_node_shared_queue.py --count 200 --submit-workers 8 --timeout-seconds 180 --report-path docs/reports/multi-node-shared-queue-report.json`
- Goal: produce a concrete two-node shared-queue proof for the current Go control plane.

## Result

- Total tasks: `200`
- Submitted by `node-a`: `100`
- Submitted by `node-b`: `100`
- Completed by `node-a`: `73`
- Completed by `node-b`: `127`
- Cross-node completions: `99`
- Duplicate `task.started`: `0`
- Duplicate `task.completed`: `0`
- Missing terminal completions: `0`

## Meaning

This run proves that two independent `bigclawd` processes can share the same SQLite-backed queue and coordinate task consumption without duplicate terminal execution in the current local topology. It is not a full distributed durability proof, but it gives the epic a concrete multi-node coordination baseline instead of relying only on single-process evidence.

In the runtime capability matrix, this shared-queue result is the current `live_proven` shared-queue proof. Subscriber takeover, stale-writer fencing, and replay coordination remain `harness_proven` or `contract_only` until the same semantics are emitted by a live multi-node run. The runtime now carries a dedicated coordinator leader-election lease for scheduler authority and failover visibility, but that lease still uses the same local SQLite durability boundary as the rest of the current proof.

## Leader Election Boundary

The coordinator lease makes scheduler ownership explicit through `scope`, `leader_id`, `lease_token`, `lease_epoch`, and expiry metadata surfaced in `/debug/status` and `/v2/reports/distributed`. That hardens failover and stale-owner fencing beyond implicit writer behavior, while still remaining a local SQLite-backed coordination mechanism rather than a broker-backed or quorum-backed control plane.

## Artifact

- `docs/reports/multi-node-shared-queue-report.json`
- `docs/reports/shared-queue-companion-summary.json`
- `docs/reports/live-validation-index.md`
- `docs/reports/cross-process-coordination-capability-surface.json`

## Parallel follow-up digests

- Cross-process coordination caveats are consolidated in `docs/reports/cross-process-coordination-boundary-digest.md`.
- Validation bundle continuation caveats are consolidated in `docs/reports/validation-bundle-continuation-digest.md`.
