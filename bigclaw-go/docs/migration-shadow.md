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
  --corpus-manifest ./examples/shadow-corpus-manifest.json \
  --report-path ./docs/reports/shadow-matrix-report.json
```

This matrix helper runs multiple shadow comparisons back to back and aggregates the
matched vs mismatched outcomes into one report. `--task-file` inputs remain the explicit
fixture-backed shadow submissions, while `--corpus-manifest` overlays an anonymized
corpus scorecard so reviewers can compare fixture coverage against corpus slices and
inspect any uncovered task shapes.

To connect the single-run compare evidence and the matrix evidence into one reviewer-facing
surface, generate the repo-native live shadow mirror scorecard:

```bash
cd bigclaw-go
python3 scripts/migration/live_shadow_scorecard.py \
  --shadow-compare-report ./docs/reports/shadow-compare-report.json \
  --shadow-matrix-report ./docs/reports/shadow-matrix-report.json \
  --output ./docs/reports/live-shadow-mirror-scorecard.json
```

The scorecard stays offline and repo-native. It summarizes parity drift and evidence
freshness across the checked-in shadow artifacts, but it is still not a live legacy-vs-Go
production traffic comparison; see `docs/reports/live-shadow-comparison-follow-up-digest.md`.

When a manifest contains approved replay payloads, add `--replay-corpus-slices` to
submit those corpus-backed slices through the same shadow matrix run. Slices without a
payload still contribute to the `corpus_coverage` scorecard and uncovered-slice summary.

## Expected output

- Primary terminal state
- Shadow terminal state
- Event timelines for both runs
- Shared `trace_id` for both runs
- Event type sequence diff and simple timeline duration comparison
- A simple diff summary indicating whether end states matched
- `corpus_coverage.shape_scorecard` comparing fixture task shapes vs anonymized corpus slices
- `corpus_coverage.uncovered_slices` listing corpus shapes that still lack fixture coverage
- `live-shadow-mirror-scorecard.json` can summarize parity drift status and evidence freshness across both checked-in reports

## Parallel follow-up digests

- Live shadow traffic comparison caveats are consolidated in `docs/reports/live-shadow-comparison-follow-up-digest.md`.
- Production corpus coverage caveats are consolidated in `docs/reports/production-corpus-migration-coverage-digest.md`.
