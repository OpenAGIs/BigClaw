# Refill Queue Metadata Drift (Local Tracker)

`docs/parallel-refill-queue.json` is the canonical refill *order* file.
When BigClaw is running on the local tracker backend, `local-issues.json` is the
canonical *state* store.

That split is intentional, but it means the queue's embedded `issues[].status`
fields can drift over time if they are not refreshed.

## What Can Drift

- `docs/parallel-refill-queue.json`:
  - `issue_order`: canonical promotion priority.
  - `issues[].status`: convenience metadata for humans and dashboards.
- `local-issues.json`:
  - `issues[].state`: source of truth for current execution state.

Drift happens when:

- issues are promoted/closed via local tracker automation, but the queue metadata
  was not rewritten.
- the queue file was edited manually and not kept in sync with local tracker
  state.

## How `bigclawctl refill` Uses State

`bigclawctl refill` always loads the queue order from `docs/parallel-refill-queue.json`.

When the backend is `local` (you passed `--local-issues local-issues.json`), the
tool prefers the full local tracker state for:

- determining whether the queue is drained (`queue_drained`)
- computing `queue_runnable`

This prevents stale queue metadata from masking terminal states (`Done`,
`Closed`, etc).

## Keeping Queue Metadata Aligned

When you want the queue file's `issues[].status` fields to match the local
tracker, run:

```bash
bash scripts/ops/bigclawctl refill \
  --apply \
  --local-issues local-issues.json \
  --sync-queue-status
```

Notes:

- `--sync-queue-status` is only meaningful for the local backend.
- the queue file is only rewritten when `--apply` is set and at least one status
  changes.
- the sync updates only `issues[].status`; it does not reorder `issue_order` or
  create missing issue records.

## When To Use It

- after closing issues in `local-issues.json` (especially when the queue starts
  showing drained/runnable anomalies in dashboards)
- after resolving a branch divergence where local tracker state was correct but
  the queue metadata was edited elsewhere
- before posting screenshots or status summaries that rely on queue metadata
  rather than the live tracker state

