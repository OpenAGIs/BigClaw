# BigClaw v5.0 Parallel Refill Queue

This file is the human-readable companion to `docs/parallel-refill-queue.json`.
It records the current BigClaw v5.0 distributed backlog slices so Symphony or a
manual operator can refill the next parallel-safe issues in a stable order.

## Trigger

- Manual one-shot refill:
  - `python3 scripts/ops/bigclaw_refill_queue.py --apply`
- Continuous refill watcher:
  - `python3 scripts/ops/bigclaw_refill_queue.py --apply --watch`
- Optional dashboard refresh after promotion:
  - `python3 scripts/ops/bigclaw_refill_queue.py --apply --watch --refresh-url http://127.0.0.1:4000/api/v1/refresh`

## Policy

- Keep at least `2` issues in `In Progress`.
- Promote only issues currently in `Backlog` or `Todo`.
- Use the queue order below as the single source of truth for refill priority.
- Every code-bearing issue must finish with GitHub push plus local/remote SHA equality verification.

## Recent batches

- Completed:
  - `OPE-206` — replicated event store / broker-backed durability planning
  - `OPE-207` — replay-safe downstream consumer contract
  - `OPE-208` — broker-backed event-log adapter contract
  - `OPE-209` — replay-safe consumer dedup ledger semantics
  - `OPE-210` — subscriber group checkpoint lease coordination
  - `OPE-211` — broker failover / replay fault-injection validation
- Active:
  - `OPE-212` — replay checkpoint compaction / retention semantics
  - `OPE-213` — durability capability matrix / backend config validation
- Standby:
  - `OPE-214` — event-log backend capability probe / control-center exposure
  - `OPE-215` — consumer dedup ledger backend abstraction / persistence key model
  - `OPE-216` — replay cursor expiry / truncated history fallback semantics
  - `OPE-217` — multi-subscriber takeover fault injection / audit evidence

## Canonical refill order

1. `OPE-210`
2. `OPE-211`
3. `OPE-212`
4. `OPE-213`
5. `OPE-214`
6. `OPE-215`
7. `OPE-216`
8. `OPE-217`
