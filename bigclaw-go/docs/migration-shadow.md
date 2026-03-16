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
so the resulting timelines stay easy to correlate in audit/event tooling. When
`--report-path` is set, the script also refreshes a normalized bundle under
`docs/reports/migration-shadow-bundle/` with:

- a canonical report copy for the latest compare or matrix run
- a lightweight `shadow-compare-summary.json` or `shadow-matrix-summary.json`
- shared top-level bundle metadata such as `bundle_path`, `summary_path`, and
  `comparison_totals`

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
matched vs mismatched outcomes into one report. Use `--bundle-dir` to override the
default `docs/reports/migration-shadow-bundle/` location when you want to keep
one run's artifacts isolated from another.

## Expected output

- Bundle metadata for canonical report, bundle report, and bundle summary paths
- Compare or matrix status under one shared `comparison_totals` vocabulary
- Primary and shadow terminal states for each comparison
- Event timelines for both runs
- Shared `trace_id` values for primary and shadow submissions
- Event type sequence diffs and simple timeline duration comparison
