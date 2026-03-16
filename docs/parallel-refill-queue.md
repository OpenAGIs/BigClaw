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
- Use `Backlog` for future standby slices; Symphony actively tracks `Todo` as runnable work.

## Recent batches

- Completed:
  - `OPE-243` — mixed workload validation drilldown normalization
  - `OPE-244` — benchmark matrix artifact normalization
  - `OPE-245` — shadow comparison artifact bundle normalization
  - `OPE-246` — lease recovery and takeover readiness digest refresh
- Active:
  - `OPE-247` — migration readiness review pack refresh
  - `OPE-250` — issue coverage and project sync evidence refresh
- Standby:
  - `OPE-251` — epic concurrency and reliability closeout refresh

## Canonical refill order

1. `OPE-247`
2. `OPE-250`
3. `OPE-251`
