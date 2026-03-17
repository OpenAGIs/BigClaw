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

- Temporary target: keep `1` issue in `In Progress` until the Linear workspace issue quota is raised.
- Promote only issues currently in `Backlog` or `Todo`.
- Use the queue order below as the single source of truth for refill priority.
- Every substantive code-bearing update must be committed and pushed to GitHub immediately, with local/remote SHA equality verification after each push.
- Shared mirror bootstrap remains mandatory so multiple Symphony issues reuse one local mirror/seed cache instead of re-downloading the repo.

## Repo Validation

- Latest repo-wide validation report: `reports/repo-wide-validation-2026-03-16.md`
- Latest mainline CI after landing `#102` / `#103` / `#104` / `#105`:
  - `256 passed` in the Python suite
  - queue / coordination / live takeover follow-up regressions passed

## Current batch

- Active:
  - none; the recycled `BIG-PAR-084` / `085` / `086` / `087` batch has landed on `main`
- Recently completed:
  - `OPE-260` — `BIG-PAR-084` live multi-node subscriber takeover proof with lease-aware checkpoints
  - `OPE-261` — `BIG-PAR-085` runtime cross-process coordination capability matrix
  - `OPE-262` — `BIG-PAR-086` enforce continuation gate in repeatable validation workflows
  - `OPE-263` — `BIG-PAR-087` live shadow mirror bundle index and parity drift rollup
  - `OPE-275` — `BIG-PAR-083` production corpus replay pack and migration coverage scorecard
- Standby:
  - no recycled slot is assigned yet; `BIG-PAR-088` remains blocked by the Linear issue quota

## Canonical refill order

- none until the next recyclable Linear slot is assigned or the workspace quota is restored
