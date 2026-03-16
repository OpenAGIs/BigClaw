# BigClaw Go Shadow Comparison

Use this helper to compare two BigClaw control-plane endpoints with the same task payload.

## Example

```bash
cd bigclaw-go
python3 scripts/migration/shadow_compare.py \
  --primary http://127.0.0.1:8080 \
  --shadow http://127.0.0.1:8081 \
  --task-file ./examples/shadow-task.json \
  --report-path ./docs/reports/shadow-compare-report.json
```

The helper now reuses one shared `trace_id` across the primary and shadow submissions,
so the resulting timelines stay easy to correlate in audit/event tooling.

## Matrix example

```bash
cd bigclaw-go
python3 scripts/migration/shadow_matrix.py \
  --primary http://127.0.0.1:8080 \
  --shadow http://127.0.0.1:8081 \
  --task-file ./examples/shadow-task.json \
  --task-file ./examples/shadow-task-budget.json \
  --task-file ./examples/shadow-task-validation.json \
  --report-path ./docs/reports/shadow-matrix-report.json
```

This matrix helper runs multiple shadow comparisons back to back and aggregates the
matched vs mismatched outcomes into one report. It remains fixture-backed evidence rather
than a live legacy-vs-Go production traffic comparison; see `docs/reports/live-shadow-comparison-follow-up-digest.md`.

## Expected output

- Primary terminal state
- Shadow terminal state
- Event timelines for both runs
- Shared `trace_id` for both runs
- Event type sequence diff and simple timeline duration comparison
- A simple diff summary indicating whether end states matched

## Parallel follow-up digests

- Live shadow traffic comparison caveats are consolidated in `docs/reports/live-shadow-comparison-follow-up-digest.md`.
