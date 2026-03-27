# BigClaw Go Migration Plan

## Goals

- Keep the old implementation available during bootstrap
- Add a compatibility layer around task protocol and issue writeback
- Support shadow traffic before cutover
- Provide a fast rollback path by task type or tenant

## Phases

1. Freeze task protocol and state machine
2. Run Go control plane in shadow mode
3. Compare queue, routing, and completion outcomes
4. Shift low-risk task classes first
5. Expand to Kubernetes and Ray-backed workloads
6. Retire legacy scheduler only after parity evidence exists

## Rollback

- Disable Go dispatcher by config flag
- Stop new leases from the Go control plane
- Hand back eligible tasks to legacy scheduler
- Keep audit trail and replay logs for every shadow run
- Treat the tenant-scoped trigger surface in `docs/reports/rollback-safeguard-follow-up-digest.md` and `docs/reports/rollback-trigger-surface.json` as the minimum rollback review gate before any tenant cutover expands
- Current rollback remains operator-driven until the safeguards in `docs/reports/rollback-safeguard-follow-up-digest.md` are implemented; the JSON trigger surface is visibility-only and does not execute rollback automatically

## Parallel Follow-up Index

- `docs/reports/parallel-follow-up-index.md` is the canonical index for the
  remaining migration, rollback, and parallel-hardening follow-up digests.
- For executor-lane validation evidence, start with
  `docs/reports/parallel-validation-matrix.md`.
- For the pytest-to-Go harness split, use
  `docs/reports/test-harness-migration-plan.md` as the canonical migration
  plan for unit, golden, and integration lanes.
