# Migration Readiness Report

## Scope

This report summarizes the current migration-readiness evidence for `OPE-185` / `BIG-GO-010`.

## Implemented surfaces

- Shadow comparison for one task via `go run ./cmd/bigclawctl automation migration shadow-compare ...`
- Shadow comparison matrix across multiple task fixtures via `go run ./cmd/bigclawctl automation migration shadow-matrix ...`
- Repo-native live shadow mirror scorecard via `go run ./cmd/bigclawctl automation migration live-shadow-scorecard`
- Repo-native live shadow bundle/index via `go run ./cmd/bigclawctl automation migration export-live-shadow-bundle`
- An anonymized corpus-manifest scorecard path via `examples/shadow-corpus-manifest.json`
- Shared `trace_id` correlation across primary/shadow runs
- JSON reports for single-run and matrix outcomes

## Evidence

- `docs/migration.md`
- `docs/migration-shadow.md`
- `cmd/bigclawctl/automation_commands.go`
- `cmd/bigclawctl/automation_commands.go`
- `go run ./cmd/bigclawctl automation migration live-shadow-scorecard`
- `go run ./cmd/bigclawctl automation migration export-live-shadow-bundle`
- `docs/reports/rollback-trigger-surface.json`
- `docs/reports/shadow-compare-report.json`
- `docs/reports/shadow-matrix-report.json`
- `docs/reports/live-shadow-mirror-scorecard.json`
- `docs/reports/live-shadow-index.md`
- `docs/reports/live-shadow-index.json`
- `docs/reports/live-shadow-drift-rollup.json`
- `docs/reports/rollback-trigger-surface.json`
- `GET /debug/status` live shadow mirror payload
- `GET /debug/status` rollback trigger payload
- `GET /v2/control-center` distributed diagnostics live shadow mirror payload
- `GET /v2/control-center` migration review rollback trigger payload
- `examples/shadow-corpus-manifest.json`

## Validation target

- Matrix should report matched terminal states and matched event-type sequences for all sample tasks before a wider cutover.

## Remaining gaps

- Still no live legacy-vs-Go production traffic comparison; see `OPE-266` / `BIG-PAR-092` in `docs/reports/live-shadow-comparison-follow-up-digest.md`.
- The live shadow mirror scorecard and bundle index are repo-native and offline; freshness comes from checked-in artifact timestamps rather than continuous mirrored traffic. Reviewers can inspect the checked-in runtime-facing mirror surface through `GET /debug/status` under `live_shadow_mirror_scorecard` and through `GET /v2/control-center` under `distributed_diagnostics.live_shadow_mirror_scorecard`.
- No tenant-scoped automated rollback trigger yet; the current trigger surface and manual rollback guardrails for `OPE-254` / `BIG-PAR-088` are documented in `docs/reports/rollback-safeguard-follow-up-digest.md` and summarized machine-readably in `docs/reports/rollback-trigger-surface.json`. Reviewers can inspect the same runtime-facing trigger payload through `GET /debug/status` under `rollback_trigger_surface` and through `GET /v2/control-center` under `distributed_diagnostics.migration_review_pack.rollback_trigger_surface`.
- Matrix now accepts anonymized corpus manifests, but the checked-in sample still defaults to local fixture tasks and requires operator-supplied corpus slices for real production-weighted evidence; see `docs/reports/production-corpus-migration-coverage-digest.md`.

## Parallel Follow-up Index

- `docs/reports/parallel-follow-up-index.md` is the canonical index for the
  remaining migration-shadow, rollback, and corpus-coverage caveats.
- Use `docs/reports/parallel-validation-matrix.md` first when the migration
  review needs the executor-lane validation evidence that sits alongside these
  follow-up tracks.
