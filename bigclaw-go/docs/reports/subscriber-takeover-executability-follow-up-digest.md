# Subscriber Takeover Executability Follow-up Digest

## Scope

This digest consolidates the remaining takeover coordination caveats after the local and live proof paths landed for `OPE-269` / `BIG-PAR-080` and `OPE-260` / `BIG-PAR-084`.

## Current Repo-Backed Evidence

- `docs/reports/multi-subscriber-takeover-validation-report.md` explains the executable local harness contract and required assertions.
- `docs/reports/multi-subscriber-takeover-validation-report.json` captures three passing local takeover scenarios with owner timelines, checkpoint transitions, and duplicate replay accounting.
- `scripts/e2e/subscriber_takeover_fault_matrix.py` generates the deterministic local harness report.
- `docs/reports/live-multi-node-subscriber-takeover-report.json` captures the live two-node shared-queue proof using the same core schema plus per-node takeover audit artifacts.
- `scripts/e2e/multi_node_shared_queue.py` now generates both the shared-queue report and the live takeover companion report in one run.
- `docs/reports/event-bus-reliability-report.md` explains how subscriber-group checkpoints, replay, and takeover evidence fit into the event-bus roadmap.
- `docs/reports/issue-coverage.md` and `docs/reports/review-readiness.md` record where takeover validation is executable today and where distributed ownership still remains bounded.
- `docs/openclaw-parallel-gap-analysis.md` tracks the remaining distributed durability and shared-queue hardening gaps.

## Reviewer Digest

- The repo now has both a deterministic local harness and a live two-node shared-queue proof that emits the same core takeover schema plus per-node audit artifacts.
- Current checkpoint fencing proves stale writers cannot advance ownership after takeover in both paths and exposes stale-write rejection counts directly in the generated reports.
- The live proof is intentionally scoped: it exercises real `bigclawd` processes and the real lease/checkpoint API, but subscriber ownership is still coordinated through a process-local lease store on one node per scenario.
- Takeover readiness is therefore reviewable as live evidence for schema parity and operational transitions, but not yet as broker-backed or shared-durable distributed ownership evidence.

## Current Blockers

- No shared durable subscriber-group coordination proof yet closes the gap between the live API-driven proof and true cross-process ownership.
- No broker-backed or replicated transport yet carries subscriber ownership across independent processes or nodes.
- Runtime task audit logs still do not emit native takeover transition events; the live proof writes per-node takeover artifacts from the harness.
- Duplicate replay candidates are still derived from checkpoint overlap windows rather than a broker-backed replay stream.

## Lightweight Consistency Check

- Keep this digest aligned with `docs/reports/multi-subscriber-takeover-validation-report.md`, `docs/reports/multi-subscriber-takeover-validation-report.json`, `scripts/e2e/subscriber_takeover_fault_matrix.py`, and `docs/reports/event-bus-reliability-report.md`.
- Keep the live companion proof references aligned with `docs/reports/live-multi-node-subscriber-takeover-report.json`, `docs/reports/live-multi-node-subscriber-takeover-artifacts/`, and `scripts/e2e/multi_node_shared_queue.py`.
- Repeat the `live schema parity exists but shared durable ownership does not` caveat anywhere takeover readiness is summarized.
