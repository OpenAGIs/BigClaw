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
- Every substantive code-bearing update must be committed and pushed to GitHub immediately, with local/remote SHA equality verification after each push.

## Recent batches

- Completed:
  - `OPE-212` — replay checkpoint compaction / retention semantics
  - `OPE-213` — durability capability matrix / backend config validation
  - `OPE-214` — event-log backend capability probe / control-center exposure
  - `OPE-215` — consumer dedup ledger backend contract
  - `OPE-216` — replay cursor expiry / truncated history fallback semantics
  - `OPE-217` — multi-subscriber takeover fault injection / audit evidence
  - `OPE-219` — auto-sync failure telemetry / PR freshness audit
  - `OPE-220` — retention watermark / replay horizon surface
  - `OPE-221` — durable consumer dedup store bootstrap
  - `OPE-222` — replicated event-log durability rollout contract
  - `OPE-223` — durable replay retention backend bootstrap
  - `OPE-224` — broker-backed event-log adapter bootstrap
  - `OPE-225` — Kubernetes / Ray / shared-queue validation bundle refresh
  - `OPE-226` — expired replay checkpoint diagnostics / reset surface
  - `OPE-227` — broker adapter dry-run capability probe
- Active:
  - `OPE-228` — checkpoint reset audit trail and operator history
  - `OPE-229` — checkpoint reset review surface in debug / control-plane payloads
- Standby:
  - `OPE-230` — checkpoint reset validation bundle refresh

## Canonical refill order

1. `OPE-228`
2. `OPE-229`
3. `OPE-230`
