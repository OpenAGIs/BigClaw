# Subscriber Takeover Executability Follow-up Digest

## Scope

This digest consolidates the remaining takeover coordination caveats after the local and live proof paths landed for `OPE-269` / `BIG-PAR-080` and `OPE-260` / `BIG-PAR-084`.

## Current Repo-Backed Evidence

- `docs/reports/multi-subscriber-takeover-validation-report.md` explains the executable local harness contract and required assertions.
- `docs/reports/multi-subscriber-takeover-validation-report.json` captures three passing local takeover scenarios with owner timelines, checkpoint transitions, and duplicate replay accounting.
- `scripts/e2e/subscriber-takeover-fault-matrix` generates the deterministic local harness report.
- `docs/reports/live-multi-node-subscriber-takeover-report.json` captures the live two-node shared-queue proof using the same core schema plus per-node takeover audit artifacts.
- `scripts/e2e/multi_node_shared_queue.py` now generates both the shared-queue report and the live takeover companion report in one run.
- `docs/reports/event-bus-reliability-report.md` explains how subscriber-group checkpoints, replay, and takeover evidence fit into the event-bus roadmap.
- `docs/reports/issue-coverage.md` and `docs/reports/review-readiness.md` record where takeover validation is executable today and where distributed ownership still remains bounded.
- `docs/openclaw-parallel-gap-analysis.md` tracks the remaining distributed durability and shared-queue hardening gaps.

## Reviewer Digest

- The repo now has both a deterministic local harness and a live two-node shared-queue proof that emits the same core takeover schema plus per-node audit artifacts sourced from runtime audit events.
- Current checkpoint fencing proves stale writers cannot advance ownership after takeover in both paths and exposes stale-write rejection counts directly in the generated reports.
- The live proof is intentionally scoped: it exercises real `bigclawd` processes and the real lease/checkpoint API on both nodes, backed by one shared SQLite lease store.
- Takeover readiness is therefore reviewable as live evidence for schema parity, operational transitions, and a shared durable scaffold, but not yet as broker-backed or replicated distributed ownership evidence.
- In short, `live schema parity exists but shared durable ownership does not` beyond the current SQLite-backed prototype.

## Current Blockers

- No broker-backed or replicated transport yet carries subscriber ownership across independent processes or nodes beyond the shared durable SQLite scaffold.
- Runtime audit logs now emit native takeover transition events, but ownership remains bounded to the current SQLite-backed scaffold until a broker-backed or replicated backend exists.
- Duplicate replay candidates are still derived from checkpoint overlap windows rather than a broker-backed replay stream.

## Lightweight Consistency Check

- Keep this digest aligned with `docs/reports/multi-subscriber-takeover-validation-report.md`, `docs/reports/multi-subscriber-takeover-validation-report.json`, `scripts/e2e/subscriber-takeover-fault-matrix`, and `docs/reports/event-bus-reliability-report.md`.
- Keep the live companion proof references aligned with `docs/reports/live-multi-node-subscriber-takeover-report.json`, `docs/reports/live-multi-node-subscriber-takeover-artifacts/`, and `scripts/e2e/multi_node_shared_queue.py`.
- Repeat the `shared durable SQLite scaffold exists but broker-backed ownership does not` caveat anywhere takeover readiness is summarized.
