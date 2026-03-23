# Refill Queue Metadata Drift (Local Tracker)

`docs/parallel-refill-queue.json` is the canonical refill *order* file.
When BigClaw is running on the local tracker backend, `local-issues.json` is the
canonical *state* store.

That split is intentional, but it means the queue's embedded convenience
metadata can drift over time if it is not refreshed.

## What Can Drift

- `docs/parallel-refill-queue.json`:
  - `issue_order`: canonical promotion priority.
  - `issues[].status`: convenience metadata for humans and dashboards.
  - `recent_batches`: convenience summary of active/completed/standby slices.
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

- reloading the tracker file on each fetch so long-lived `refill --watch` runs
  observe external tracker edits
- determining whether the queue is drained (`queue_drained`)
- computing `queue_runnable`
- refreshing `recent_batches` from the live local tracker map

This prevents stale queue metadata from masking terminal states (`Done`,
`Closed`, etc).

## Keeping Queue Metadata Aligned

When you want the queue file's embedded metadata to match the local tracker,
run:

```bash
bash scripts/ops/bigclawctl refill \
  --apply \
  --local-issues local-issues.json \
  --sync-queue-status
```

Notes:

- `--sync-queue-status` is only meaningful for the local backend.
- the queue file is only rewritten when `--apply` is set and either
  `issues[].status` or `recent_batches` is stale.
- the sync updates embedded metadata only; it does not reorder `issue_order` or
  create missing issue records.
- current command payloads also report `recent_batches_synced`,
  `recent_batches_updated`, and `recent_batches_written` so callers can see when
  the summary metadata was refreshed.

## When To Use It

- after closing issues in `local-issues.json` (especially when the queue starts
  showing drained/runnable anomalies in dashboards)
- after resolving a branch divergence where local tracker state was correct but
  the queue metadata was edited elsewhere
- before posting screenshots or status summaries that rely on queue metadata
  rather than the live tracker state
- when testing `refill --watch`, because the local backend now reloads
  `local-issues.json` on each fetch and should observe tracker edits made by
  another process without restarting the watch loop
