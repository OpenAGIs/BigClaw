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
- Keep `local-issues.json` mutations serialized; multiple concurrent writers can
  cause lost updates even when the file stays valid.
- Prefer absolute repo roots in hook invocations so `--local-issues ../local-issues.json`
  resolves to the workspace tracker file (not a nested `bigclaw-go/local-issues.json`).

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

