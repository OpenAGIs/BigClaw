# Live Shadow Comparison Follow-up Digest

## Scope

This digest consolidates the remaining live shadow-traffic comparison caveats for `OPE-266` / `BIG-PAR-092`.

## Current Repo-Backed Evidence

- `docs/reports/migration-readiness-report.md` captures the currently shipped shadow-compare and shadow-matrix evidence.
- `docs/migration-shadow.md` documents the single-run and matrix helper workflows.
- `docs/reports/shadow-compare-report.json` captures one shared-`trace_id` comparison sample.
- `docs/reports/shadow-matrix-report.json` captures the multi-fixture comparison matrix.
- `docs/reports/live-shadow-mirror-scorecard.json` connects the checked-in compare and matrix artifacts into one repo-native parity drift and freshness scorecard.
- `docs/reports/migration-plan-review-notes.md` records the current shadow-before-cutover design boundary.

## Reviewer Digest

- The repo-native live shadow mirror scorecard now summarizes parity drift and evidence freshness across the checked-in compare and matrix artifacts.
- Current shadow evidence is still fixture-backed and repo-local; there is no live legacy-vs-Go production traffic comparison.
- The existing compare, matrix, and scorecard surfaces prove timeline / terminal-state parity on sample tasks, not on mirrored production requests.
- Shared `trace_id` correlation makes local audit review easier, but it is not the same as a real shadow ingress or traffic duplication path.
- Current cutover confidence therefore comes from controlled samples and live smoke evidence, not from continuous shadow traffic.

## Current Blockers

- No always-on live shadow request duplication path exists yet.
- No production ingress mirror or tenant-scoped shadow routing control exists yet.
- No real legacy-vs-Go cutover evidence exists from mirrored live traffic.
- The current freshness signal is derived from checked-in artifact timestamps rather than continuous operational telemetry.

## Lightweight Consistency Check

- Keep this digest aligned with `docs/reports/migration-readiness-report.md` and `docs/migration-shadow.md`.
- Repeat the `repo-native live shadow mirror scorecard`, `no live legacy-vs-Go production traffic comparison`, and `fixture-backed shadow evidence only` caveats anywhere migration readiness is summarized.
- When live shadow traffic lands, update this digest, `docs/reports/review-readiness.md`, and `docs/reports/issue-coverage.md` together.
