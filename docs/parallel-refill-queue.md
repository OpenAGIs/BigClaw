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
  - `python3 scripts/ops/bigclaw_refill_queue.py --apply --watch --refresh-url http://127.0.0.1:4001/api/v1/refresh`

## Policy

- Current target: keep `4` issues in `In Progress` while the current batch is parallel-safe.
- Promote only issues currently in `Backlog` or `Todo`.
- Use the queue order below as the single source of truth for refill priority.
- Every substantive code-bearing update must be committed and pushed to GitHub immediately, with local/remote SHA equality verification after each push.
- Shared mirror bootstrap remains mandatory so multiple Symphony issues reuse one local mirror/seed cache instead of re-downloading the repo.
- Linear free issue quota still blocks net-new issues, so the active batch uses recycled done issue slots.

## Current batch

- Active:
  - `OPE-234` — `BIG-PAR-100` replace durability rollout placeholder with stub failover evidence
  - `OPE-231` — `BIG-PAR-101` broker failover review pack links in distributed export and control center
  - `OPE-227` — `BIG-PAR-102` local stub broker event-log backend path in bootstrap
  - `OPE-230` — `BIG-PAR-103` checkpoint fencing and retention proof summary from broker stub matrix
- Ready to promote:
  - none; current batch is already fully active

## Canonical refill order

1. `OPE-234`
2. `OPE-231`
3. `OPE-227`
4. `OPE-230`
