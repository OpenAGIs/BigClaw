# Live Shadow Mirror Index

- Latest run: `20260313T085655Z`
- Generated at: `2026-03-17T02:35:33.529497Z`
- Status: `parity-ok`
- Severity: `none`
- Bundle: `docs/reports/live-shadow-runs/20260313T085655Z`
- Summary JSON: `docs/reports/live-shadow-runs/20260313T085655Z/summary.json`

## Latest bundle artifacts

- Shadow compare report: `docs/reports/live-shadow-runs/20260313T085655Z/shadow-compare-report.json`
- Shadow matrix report: `docs/reports/live-shadow-runs/20260313T085655Z/shadow-matrix-report.json`
- Parity scorecard: `docs/reports/live-shadow-runs/20260313T085655Z/live-shadow-mirror-scorecard.json`
- Rollback trigger surface: `docs/reports/live-shadow-runs/20260313T085655Z/rollback-trigger-surface.json`

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
- Rollback trigger surface status: `manual-review-required`
- Rollback automation boundary: `manual_only`
- Rollback trigger distinctions: `{'blockers': 3, 'warnings': 1, 'manual_only_paths': 2}`

## Parity drift rollup

- Status: `parity-ok`
- Latest run: `20260313T085655Z`
- Highest severity: `none`
- Drift-detected runs in window: `0`
- Stale runs in window: `0`

## Workflow closeout commands

- `cd bigclaw-go && go run ./cmd/bigclawctl migration live-shadow-scorecard --repo .`
- `cd bigclaw-go && python3 scripts/migration/export_live_shadow_bundle.py`
- `cd bigclaw-go && go test ./internal/regression -run TestRollbackDocsStayAligned`
- `git push origin <branch> && git log -1 --stat`

## Recent bundles

- `20260313T085655Z` · `parity-ok` · `none` · `2026-03-17T02:35:33.529497Z` · `docs/reports/live-shadow-runs/20260313T085655Z`

## Linked migration docs

- `docs/migration-shadow.md` Shadow helper workflow and bundle generation steps.
- `docs/reports/migration-readiness-report.md` Migration readiness summary linked to the shadow bundle.
- `docs/reports/migration-plan-review-notes.md` Review notes tied to the shadow bundle index.
- `docs/reports/rollback-trigger-surface.json` Machine-readable rollback blockers, warnings, and manual-only paths linked from the shadow bundle.

## Parallel Follow-up Index

- `docs/reports/parallel-follow-up-index.md` is the canonical index for the
  remaining live-shadow, rollback, and corpus-coverage follow-up digests.
- Use `docs/reports/parallel-validation-matrix.md` first when a shadow review
  needs the checked-in local/Kubernetes/Ray validation entrypoint alongside the
  shadow evidence bundle.
