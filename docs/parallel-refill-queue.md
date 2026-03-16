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

- Temporary target: keep `1` issue in `In Progress` until the Linear workspace issue quota is raised.
- Promote only issues currently in `Backlog` or `Todo`.
- Use the queue order below as the single source of truth for refill priority.
- Every substantive code-bearing update must be committed and pushed to GitHub immediately, with local/remote SHA equality verification after each push.
- Shared mirror bootstrap remains mandatory so multiple Symphony issues reuse one local mirror/seed cache instead of re-downloading the repo.

## Repo Validation

- Latest repo-wide validation report: `reports/repo-wide-validation-2026-03-16.md`
- Latest findings:
  - `233 passed` in the Python suite
  - `ruff check` passed
  - `go test ./...` passed in `bigclaw-go`
  - deprecated `datetime.utcnow()` usage was cleaned up in `src/bigclaw/reports.py`

## Current batch

- Active:
  - `OPE-275` — production corpus replay pack and migration coverage scorecard
- Standby (planned but blocked by Linear issue quota):
  - `BIG-PAR-084` — executable subscriber takeover harness with lease-aware checkpoints
  - `BIG-PAR-085` — cross-process coordination capability surface
  - `BIG-PAR-086` — rolling validation bundle continuation scorecard
  - `BIG-PAR-087` — live shadow traffic mirror and parity drift scorecard
  - `BIG-PAR-088` — tenant-scoped rollback guardrails and trigger surface

## Canonical refill order

1. `OPE-275`
