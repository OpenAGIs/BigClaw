# BigClaw v5.2 Broker Adapter Bootstrap Refill Queue

This file is the human-readable companion to `docs/parallel-refill-queue.json`.
It records the current BigClaw v5.2 broker-bootstrap slices so Symphony or a
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
- Linear free issue quota still blocks net-new issues, so the active batch uses recycled completed issue slots.

## Current batch

- Active:
  - `OPE-260` — `BIG-BRK-201` broker bootstrap readiness surface
  - `OPE-261` — `BIG-BRK-202` broker config completeness diagnostics
- Ready to promote:
  - `OPE-263` — `BIG-BRK-203` broker runtime gate and fail-closed posture
  - `OPE-264` — `BIG-BRK-204` broker review bundle unification

## Previous completed batch

- `OPE-5` — `BIG-DUR-101` publish acknowledgement outcome ledger
- `OPE-12` — `BIG-DUR-102` durable sequence bridge for provider offsets
- `OPE-21` — `BIG-DUR-103` provider-backed retention watermark and expiry surface
- `OPE-225` — `BIG-DUR-104` provider-backed live handoff isolation proof

## Canonical refill order

1. `OPE-260`
2. `OPE-261`
3. `OPE-263`
4. `OPE-264`
