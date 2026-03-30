# BigClaw Go Shadow Comparison

Use this helper to compare two BigClaw control-plane endpoints with the same task payload.

## Example

```bash
cd bigclaw-go
go run ./cmd/bigclawctl automation migration shadow-compare \
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
go run ./cmd/bigclawctl automation migration shadow-matrix \
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

To package the checked-in shadow artifacts into a repeatable reviewer bundle and refresh
the parity drift rollup, export the live shadow bundle/index:

```bash
cd bigclaw-go
python3 scripts/migration/export_live_shadow_bundle.py
```

This exporter copies the latest compare, matrix, scorecard, and rollback trigger summary artifacts into
`docs/reports/live-shadow-runs/<run-id>/`, refreshes `docs/reports/live-shadow-summary.json`,
and updates `docs/reports/live-shadow-index.md`, `docs/reports/live-shadow-index.json`, and
`docs/reports/live-shadow-drift-rollup.json` for reviewer navigation.

The checked-in bundle summary is also exposed through `GET /debug/status` as
`live_shadow_mirror_scorecard` and through `GET /v2/control-center` as
`distributed_diagnostics.live_shadow_mirror_scorecard`, so reviewers can inspect parity drift,
freshness, and report links without reopening the standalone migration bundle first.

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
- `live-shadow-index.md` can summarize the latest bundled shadow artifacts and reviewer navigation paths
- `live-shadow-drift-rollup.json` can summarize freshness and mismatch severity across recent bundled shadow runs
- `rollback-trigger-surface.json` can distinguish tenant-scoped blockers, warnings, and manual-only rollback paths without claiming automated rollback execution

## Parallel Follow-up Index

- `docs/reports/parallel-follow-up-index.md` is the canonical index for the
  remaining migration-shadow and parallel-hardening follow-up digests.
- Use `docs/reports/parallel-validation-matrix.md` first when a migration-shadow
  review needs the checked-in local/Kubernetes/Ray validation entrypoint.
- Production corpus coverage caveats remain tracked in
  `docs/reports/production-corpus-migration-coverage-digest.md`.
