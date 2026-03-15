# OpenClaw Parallel Gap Analysis

## Replay and checkpoint durability track

The current BigClaw Go event plane has replay-capable APIs, but the implementation remains process-local and does not yet define how replay history ages out without breaking subscriber recovery.

### Closed baseline

- `OPE-199` introduced a durable-event-log direction, but the current checkout still exposes replay primarily through recorder history and in-memory bus subscriptions.
- `OPE-203` added subscriber checkpoint and resume semantics at the issue-planning level, while `OPE-205` tightened monotonic checkpoint expectations.
- `OPE-210` defined subscriber-group lease coordination so stale writers cannot move shared progress backward.

### Active gap closed by `OPE-212`

- Replay history needs a retention contract that preserves a contiguous replay window.
- Subscriber checkpoints need an explicit validity rule once older history is compacted away.
- Resume failures caused by aged-out checkpoints must produce diagnostics instead of silently fast-forwarding consumers.

### Remaining follow-on slices

- `OPE-213`: define backend capability matrix and config validation for durable event-log providers.
- `OPE-214`: expose backend capability probe and operator-facing retention support visibility.
- `OPE-215`: define dedup-ledger backend/key semantics for replay-safe consumers.
- `OPE-216`: define the concrete expired-cursor and truncated-history fallback surface.
- `OPE-217`: define multi-subscriber takeover fault-injection and audit evidence.

## Outcome

`OPE-212` should be treated as the contract slice that prevents future durable backends from inventing incompatible compaction behavior. The implementation bar for later slices is now:

- compact only on prefix boundaries,
- reject expired checkpoints explicitly,
- keep checkpoint cleanup separate from history retention,
- surface retention watermarks for operators and future automation.
