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

This run proves that two independent `bigclawd` processes can share the same SQLite-backed queue and coordinate task consumption without duplicate terminal execution in the current local topology. It is not a full leader-election system, but it gives the epic a concrete multi-node coordination proof instead of relying only on single-process evidence.

## Current limits and follow-up links

- No dedicated leader-election layer exists yet; this remains a two-node shared-SQLite coordination proof rather than durable cross-node control-plane ownership.
- Shared multi-node subscriber-group checkpoint coordination still needs a durable backend; the current event-bus limitation and rollout boundary are summarized in `docs/reports/event-bus-reliability-report.md`.
- Multi-subscriber takeover fault injection is defined as a planning-ready matrix in `docs/reports/multi-subscriber-takeover-validation-report.md` and `docs/reports/multi-subscriber-takeover-validation-report.json`, but it is not executable until lease-aware checkpoint ownership exists.
- Reviewer-facing summary surfaces in `docs/reports/review-readiness.md` and `docs/reports/issue-coverage.md` should continue to point back to this report when describing the remaining coordination hardening gap.

## Artifact

- `docs/reports/multi-node-shared-queue-report.json`
