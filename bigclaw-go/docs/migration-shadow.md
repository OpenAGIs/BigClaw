# BigClaw Go Shadow Comparison

Use this helper to compare two BigClaw control-plane endpoints with the same task payload.

## Example

The checked-in single-run comparison artifact lives at
`docs/reports/shadow-compare-report.json`. It reuses one shared `trace_id`
across the primary and shadow submissions, so the resulting timelines stay easy
to correlate in audit/event tooling.

## Matrix example

The checked-in matrix artifact lives at `docs/reports/shadow-matrix-report.json`
and summarizes the primary/shadow outcomes across the repo fixture set.

The matrix evidence aggregates matched vs mismatched outcomes into one report.
`examples/shadow-corpus-manifest.json` still overlays an anonymized corpus
scorecard so reviewers can compare fixture coverage against corpus slices and
inspect any uncovered task shapes.

To connect the single-run compare evidence and the matrix evidence into one reviewer-facing
surface, generate the repo-native live shadow mirror scorecard:

The checked-in scorecard artifact lives at
`docs/reports/live-shadow-mirror-scorecard.json`.

The scorecard stays offline and repo-native. It summarizes parity drift and evidence
freshness across the checked-in shadow artifacts, but it is still not a live legacy-vs-Go
production traffic comparison; see `docs/reports/live-shadow-comparison-follow-up-digest.md`.

To package the checked-in shadow artifacts into a repeatable reviewer bundle and refresh
the parity drift rollup, export the live shadow bundle/index:

The checked-in bundle/index artifacts live under
`docs/reports/live-shadow-runs/<run-id>/`, `docs/reports/live-shadow-summary.json`,
`docs/reports/live-shadow-index.md`, `docs/reports/live-shadow-index.json`, and
`docs/reports/live-shadow-drift-rollup.json`.

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
