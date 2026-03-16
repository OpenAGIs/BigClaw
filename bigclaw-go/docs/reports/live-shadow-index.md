# Live Shadow Mirror Index

- Latest run: `20260313T085655Z`
- Generated at: `2026-03-16T15:54:36.830980Z`
- Status: `parity-ok`
- Severity: `none`
- Bundle: `docs/reports/live-shadow-runs/20260313T085655Z`
- Summary JSON: `docs/reports/live-shadow-runs/20260313T085655Z/summary.json`

## Latest bundle artifacts

- Shadow compare report: `docs/reports/live-shadow-runs/20260313T085655Z/shadow-compare-report.json`
- Shadow matrix report: `docs/reports/live-shadow-runs/20260313T085655Z/shadow-matrix-report.json`
- Parity scorecard: `docs/reports/live-shadow-runs/20260313T085655Z/live-shadow-mirror-scorecard.json`

## Latest run summary

- Compare trace: `shadow-compare-sample`
- Matrix trace count: `3`
- Evidence runs: `4`
- Parity-ok entries: `4`
- Drift-detected entries: `0`
- Matrix total: `3`
- Matrix mismatched: `0`
- Fresh inputs: `2`
- Stale inputs: `0`

## Parity drift rollup

- Status: `parity-ok`
- Latest run: `20260313T085655Z`
- Highest severity: `none`
- Drift-detected runs in window: `0`
- Stale runs in window: `0`

## Workflow closeout commands

- `cd bigclaw-go && python3 scripts/migration/live_shadow_scorecard.py --pretty`
- `cd bigclaw-go && python3 scripts/migration/export_live_shadow_bundle.py`
- `pytest tests/test_live_shadow_scorecard.py tests/test_live_shadow_bundle.py`
- `git push origin <branch> && git log -1 --stat`

## Recent bundles

- `20260313T085655Z` · `parity-ok` · `none` · `2026-03-16T15:54:36.830980Z` · `docs/reports/live-shadow-runs/20260313T085655Z`

## Linked migration docs

- `docs/migration-shadow.md` Shadow helper workflow and bundle generation steps.
- `docs/reports/migration-readiness-report.md` Migration readiness summary linked to the shadow bundle.
- `docs/reports/migration-plan-review-notes.md` Review notes tied to the shadow bundle index.

## Parallel follow-up digests

- `docs/reports/live-shadow-comparison-follow-up-digest.md` Live shadow traffic comparison caveats are consolidated here.
