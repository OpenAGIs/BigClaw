# Local Tracker Automation Notes

BigClaw uses the repo-native `local-issues.json` as the source of truth for
planning, state transitions, and closeout comments.

## `bigclawctl local-issues ensure`

`bigclawctl local-issues ensure` is intended as a safety net for automation and
workflow hooks so a workspace never runs without a corresponding tracker entry.

Behavior:
- If the issue already exists, `ensure` returns `action=exists` and makes no
  changes by default.
- If the issue is missing, `ensure` creates a new local tracker record and
  returns `action=created`.
- If `--set-state-if-exists` is set, `ensure` will also update the state for an
  existing issue record.

Guidance:
- Use `ensure` in hooks (`after_create`, `before_run`) to backfill missing
  tracker records.
- Avoid using `--set-state-if-exists` outside of explicit claim flows (for
  example, `after_create`) to prevent accidentally resurrecting `Done` issues
  as `In Progress`.
- `bigclawctl` local-tracker mutations now take an explicit `local-issues.json.lock`
  file with bounded retry before rewriting the store, so overlapping hook/worker
  writes do not silently drop earlier comments or state changes.
- Keep `local-issues.json` mutations short-lived even with the lock in place; the
  lock prevents overlapping writes, but long-running critical sections still
  serialize every other tracker update behind them.
- Prefer absolute repo roots in hook invocations so `--local-issues ../local-issues.json`
  resolves to the workspace tracker file (not a nested `bigclaw-go/local-issues.json`).

## Rate-limit fallback

If Symphony or Codex child-agent orchestration starts returning
`429 Too Many Requests`:

- stop spawning new child agents for the current batch;
- keep the currently claimed issue in `In Progress` and continue the slice
  locally in the existing workspace;
- lower the workflow concurrency caps before re-enabling broad fanout so the
  next polling cycle does not immediately re-trigger the same rate limit;
- prefer one fresh issue branch at a time until GitHub sync, local validation,
  and tracker updates are stable again.

Examples:

```bash
# Create-if-missing without mutating existing state.
bash scripts/ops/bigclawctl local-issues ensure \
  --local-issues local-issues.json \
  --identifier BIG-PAR-999 \
  --state "In Progress" \
  --json

# Ensure and claim (only when the intent is to move the issue into active work).
bash scripts/ops/bigclawctl local-issues ensure \
  --local-issues local-issues.json \
  --identifier BIG-PAR-999 \
  --state "In Progress" \
  --set-state-if-exists \
  --json
```
