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
- Linear workspace issue quota still blocks net-new issues, so this batch uses recycled done issue slots.

## Current batch

- Active:
  - `OPE-272` — `BIG-PAR-096` deterministic broker failover scenario runner and stub backend
  - `OPE-273` — `BIG-PAR-097` broker durability rollout scorecard and debug summary
  - `OPE-274` — `BIG-PAR-098` broker validation lane scaffold in live bundle export
  - `OPE-271` — `BIG-PAR-099` rollout rollback policy gate across local Kubernetes and Ray
- Ready to promote:
  - none; current batch is already fully active

## Canonical refill order

1. `OPE-272`
2. `OPE-273`
3. `OPE-274`
4. `OPE-271`
