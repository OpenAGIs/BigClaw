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
  - `OPE-260` — `BIG-PAR-104` ambiguous publish outcome proof summary from broker stub matrix
  - `OPE-261` — `BIG-PAR-105` broker stub live fanout isolation evidence pack
  - `OPE-263` — `BIG-PAR-107` durable dedupe readiness scorecard for HTTP and broker event logs
  - `OPE-264` — `BIG-PAR-108` delivery acknowledgement readiness surface for event log backends
- Ready to promote:
  - none; current batch is already fully active

## Canonical refill order

1. `OPE-260`
2. `OPE-261`
3. `OPE-263`
4. `OPE-264`
