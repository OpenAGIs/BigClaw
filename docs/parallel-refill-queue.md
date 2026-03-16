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

- Keep at least `6` issues in `In Progress` for the current high-throughput batch.
- Promote only issues currently in `Backlog` or `Todo`.
- Use the queue order below as the single source of truth for refill priority.
- Every substantive code-bearing update must be committed and pushed to GitHub immediately, with local/remote SHA equality verification after each push.
- Use `Backlog` for future standby slices; Symphony actively tracks `Todo` as runnable work.

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
  - `OPE-228` — checkpoint reset audit trail and operator history
  - `OPE-229` — checkpoint reset review surface in debug / control-plane payloads
  - `OPE-230` — checkpoint reset validation bundle refresh
  - `OPE-231` — checkpoint reset distributed report / export integration
  - `OPE-232` — checkpoint reset rollout evidence / review-pack refresh
  - `OPE-233` — Ray executor live readiness rollup in distributed diagnostics
  - `OPE-234` — replicated event-log rollout readiness summary
  - `OPE-235` — shared-queue takeover evidence bundle refresh
  - `OPE-236` — kubernetes executor live readiness rollup in distributed diagnostics
  - `OPE-237` — normalize live-validation evidence bundle index
  - `OPE-238` — distributed rollout review-pack refresh
  - `OPE-239` — broker failover evidence bundle export normalization
  - `OPE-240` — durability backend comparison readiness report refresh
  - `OPE-241` — distributed closure readiness scorecard refresh
  - `OPE-242` — control-center takeover history digest
  - `OPE-243` — mixed workload validation drilldown normalization
  - `OPE-244` — benchmark matrix artifact normalization
  - `OPE-245` — shadow comparison artifact bundle normalization
  - `OPE-246` — lease recovery and takeover readiness digest refresh
  - `OPE-247` — migration readiness review pack refresh
  - `OPE-250` — issue coverage and project sync evidence refresh
  - `OPE-251` — epic concurrency and reliability closeout refresh
  - `OPE-252` — worker lifecycle and state-machine closeout digest refresh
  - `OPE-253` — control-plane observability evidence refresh
  - `OPE-254` — long-duration soak and benchmark closeout refresh
  - `OPE-255` — operations foundation evidence alignment refresh
  - `OPE-256` — scheduler policy and routing closeout refresh
  - `OPE-257` — review matrix and closeout navigation refresh
  - `OPE-258` — remaining hardening gap register refresh
  - `OPE-259` — follow-up roadmap and gap-analysis refresh
  - `OPE-260` — scale validation follow-up digest
  - `OPE-261` — distributed coordination hardening digest
  - `OPE-262` — event delivery semantics follow-up digest
  - `OPE-263` — retention and external-store follow-up digest
- Active:
  - `OPE-264` — observability tracing backend follow-up digest
  - `OPE-265` — telemetry pipeline controls follow-up digest
  - `OPE-266` — live shadow traffic comparison follow-up digest
  - `OPE-267` — rollback safeguard follow-up digest
  - `OPE-268` — production corpus migration coverage digest
  - `OPE-269` — subscriber takeover executability follow-up digest
- Standby:
  - `OPE-270` — cross-process coordination boundary digest
  - `OPE-271` — validation bundle continuation digest

## Canonical refill order

1. `OPE-264`
2. `OPE-265`
3. `OPE-266`
4. `OPE-267`
5. `OPE-268`
6. `OPE-269`
7. `OPE-270`
8. `OPE-271`
