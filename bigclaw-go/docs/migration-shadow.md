# BigClaw Go Shadow Comparison

The ad hoc Python shadow-comparison helpers and sample fixture payloads were retired in
`BIG-GO-136`. This page now documents the checked-in migration evidence that remains
available for reviewer inspection.

## Archived evidence set

- `docs/reports/shadow-compare-report.json` captures one shared-`trace_id` compare sample.
- `docs/reports/shadow-matrix-report.json` captures the multi-input parity matrix and the
  corpus-coverage overlay distilled from the archived fixture/demo set.
- `docs/reports/live-shadow-mirror-scorecard.json` summarizes parity drift and freshness
  across the checked-in compare and matrix reports.
- `docs/reports/live-shadow-index.md`, `docs/reports/live-shadow-index.json`, and
  `docs/reports/live-shadow-summary.json` preserve the reviewer-facing bundle and drift rollup.

The scorecard stays offline and repo-native. It summarizes parity drift and evidence
freshness across the checked-in shadow artifacts, but it is still not a live legacy-vs-Go
production traffic comparison; see `docs/reports/live-shadow-comparison-follow-up-digest.md`.

The checked-in bundle summary is also exposed through `GET /debug/status` as
`live_shadow_mirror_scorecard` and through `GET /v2/control-center` as
`distributed_diagnostics.live_shadow_mirror_scorecard`, so reviewers can inspect parity drift,
freshness, and report links without reopening the standalone migration bundle first.

## Expected output

- Primary terminal state
- Shadow terminal state
- Event timelines for both runs
- Shared `trace_id` for both runs
- Event type sequence diff and simple timeline duration comparison
- A simple diff summary indicating whether end states matched
- `corpus_coverage.shape_scorecard` comparing archived task-shape coverage vs anonymized corpus slices
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
