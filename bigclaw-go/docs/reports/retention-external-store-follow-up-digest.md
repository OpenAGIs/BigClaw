# Retention and External-Store Follow-up Digest

## Scope

This digest consolidates the remaining replay-retention, external-store validation, and replicated-durability follow-up work after the latest closeout wave for `OPE-263` / `BIG-PAR-074`.

It is intentionally limited to repo-native evidence and consistency checks. It does not claim new runtime behavior beyond what the linked reports and code already implement.

## Current repo-backed position

- Replay retention watermarks and replay horizon diagnostics are exposed through API/debug surfaces and event-log service reporting.
- SQLite-backed event logs now persist trimmed replay boundaries across restarts when a retention window is configured, so retained replay state survives process reboot.
- Expired durable checkpoint resumes now fail closed with explicit reset guidance, and checkpoint reset history is persisted for operator review.
- Durable consumer dedup bootstrap exists only for SQLite-backed persistence; broader shared external-store coverage is still incomplete.

## Stable evidence map

- Replay retention and expired-cursor semantics: `docs/reports/replay-retention-semantics-report.md`
- Event-bus durability shape and backend roadmap: `docs/reports/event-bus-reliability-report.md`
- Reviewer entrypoint for current hardening caveats: `docs/reports/review-readiness.md`
- MVP coverage and remaining honest-closeout gaps: `docs/reports/issue-coverage.md`
- Epic-level OpenClaw comparison and remaining distributed gaps: `docs/openclaw-parallel-gap-analysis.md`

## Follow-up digest

### Replay retention

- The retained replay window is now an explicit contract: resumability depends on the oldest retained cursor, not just whether a checkpoint payload is well formed.
- Expired checkpoints must return diagnostics and reset guidance instead of silently resuming from a later point.
- SQLite now preserves trimmed replay boundaries across restarts, which closes the gap for single-node durable retention evidence.
- Memory-backed deployments remain bounded by in-process history, so replay guarantees still disappear on restart when no durable log is configured.

### External-store validation

- SQLite is the only concrete durable external store evidenced in this checkout for replay retention and dedup persistence.
- Shared service or replicated backends remain planned rather than implemented, so higher-scale external-store validation is still pending beyond the SQLite-backed path.
- Review surfaces should keep describing current evidence as SQLite-backed durability, not generic distributed durability.

### Broker and quorum future work

- Replicated broker or quorum-backed event-log adapters remain future work.
- The rollout contract is documented, including publish acknowledgement, shared replay/checkpoint sequencing, retention-boundary visibility, and failover expectations.
- No current repo evidence supports claiming cross-process replicated replay durability, partitioned topic coordination, or exactly-once delivery semantics.

## Reviewer-facing conclusion

- BigClaw now has explicit replay-retention diagnostics and a durable single-node SQLite path.
- BigClaw does not yet have a validated shared or replicated external event-log backend.
- BigClaw remains replay-safe rather than exactly-once; downstream idempotency and broader durable-store coverage are still follow-up work.

## Consistency checklist

- `docs/reports/replay-retention-semantics-report.md` is the canonical source for retention and expired-cursor semantics.
- `docs/reports/event-bus-reliability-report.md` is the canonical source for backend capability, SQLite durability, and replicated-backend future work.
- `docs/reports/review-readiness.md`, `docs/reports/issue-coverage.md`, and `docs/openclaw-parallel-gap-analysis.md` should point reviewers back to this digest instead of restating divergent caveats.
