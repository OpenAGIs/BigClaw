# Migration Readiness Report

## Scope

This report summarizes the current migration-readiness evidence for `OPE-185` / `BIG-GO-010`.

## Implemented surfaces

- Shadow comparison for one task via `scripts/migration/shadow_compare.py`
- Shadow comparison matrix across multiple task fixtures via `scripts/migration/shadow_matrix.py`
- Repo-native live shadow mirror scorecard via `scripts/migration/live_shadow_scorecard.py`
- Repo-native live shadow bundle/index via `scripts/migration/export_live_shadow_bundle.py`
- An anonymized corpus-manifest scorecard path via `examples/shadow-corpus-manifest.json`
- Shared `trace_id` correlation across primary/shadow runs
- JSON reports for single-run and matrix outcomes

## Evidence

- `docs/migration.md`
- `docs/migration-shadow.md`
- `scripts/migration/shadow_compare.py`
- `scripts/migration/shadow_matrix.py`
- `scripts/migration/live_shadow_scorecard.py`
- `scripts/migration/export_live_shadow_bundle.py`
- `docs/reports/rollback-trigger-surface.json`
- `docs/reports/shadow-compare-report.json`
- `docs/reports/shadow-matrix-report.json`
- `docs/reports/live-shadow-mirror-scorecard.json`
- `docs/reports/live-shadow-index.md`
- `docs/reports/live-shadow-index.json`
- `docs/reports/live-shadow-drift-rollup.json`
- `docs/reports/rollback-trigger-surface.json`
- `examples/shadow-corpus-manifest.json`

## Validation target

- Matrix should report matched terminal states and matched event-type sequences for all sample tasks before a wider cutover.

## Remaining gaps

- Still no live legacy-vs-Go production traffic comparison; see `docs/reports/live-shadow-comparison-follow-up-digest.md`.
- The live shadow mirror scorecard and bundle index are repo-native and offline; freshness comes from checked-in artifact timestamps rather than continuous mirrored traffic.
- No tenant-scoped automated rollback trigger yet; the current trigger surface and manual rollback guardrails are documented in `docs/reports/rollback-safeguard-follow-up-digest.md` and summarized machine-readably in `docs/reports/rollback-trigger-surface.json`.
- Matrix now accepts anonymized corpus manifests, but the checked-in sample still defaults to local fixture tasks and requires operator-supplied corpus slices for real production-weighted evidence; see `docs/reports/production-corpus-migration-coverage-digest.md`.

## Parallel follow-up digests

- `OPE-266` / `BIG-PAR-092` — `docs/reports/live-shadow-comparison-follow-up-digest.md`
- `OPE-254` / `BIG-PAR-088` — `docs/reports/rollback-safeguard-follow-up-digest.md` and `docs/reports/rollback-trigger-surface.json`.
- `OPE-268` / `BIG-PAR-079` — `docs/reports/production-corpus-migration-coverage-digest.md`
