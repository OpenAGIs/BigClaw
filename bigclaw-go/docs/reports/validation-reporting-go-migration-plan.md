# Validation / Reporting Go Migration Plan

## Scope

This plan records the `BIG-GO-907` migration slice for validation bundle, reports, and evaluation-adjacent tooling that still relies on non-Go logic.

## First Batch Implemented

- `go run ./cmd/bigclawctl migration validation-continuation-scorecard`
  Replaces the Python-only continuation scorecard refresh path for `docs/reports/validation-bundle-continuation-scorecard.json`.
- `go run ./cmd/bigclawctl migration validation-continuation-policy-gate`
  Replaces the Python-only continuation policy gate path for `docs/reports/validation-bundle-continuation-policy-gate.json`.
- `bigclaw-go/docs/e2e-validation.md`
  Validation guidance now points the primary operator workflow at the Go entrypoints above.

## Remaining Migration Queue

1. Migrate `scripts/e2e/export_validation_bundle.py` into Go so bundle export, README refresh, and canonical summary/index updates stop depending on Python orchestration.
2. Migrate `scripts/migration/export_live_shadow_bundle.py` into Go so the shadow bundle/index path matches the runtime-facing Go API surfaces already checked in.
3. Migrate evaluation-heavy helpers that still summarize checked-in report corpora through Python, especially where output is consumed by `docs/reports/*` review packs and `internal/api/*` surfaces.
4. Collapse compatibility wrappers so Python scripts become thin delegators or are removed after parity validation.

## Validation Commands

```bash
cd bigclaw-go
go test ./internal/migration ./cmd/bigclawctl
go run ./cmd/bigclawctl migration validation-continuation-scorecard \
  --output docs/reports/validation-bundle-continuation-scorecard.json
go run ./cmd/bigclawctl migration validation-continuation-policy-gate \
  --scorecard docs/reports/validation-bundle-continuation-scorecard.json \
  --output docs/reports/validation-bundle-continuation-policy-gate.json
```

## Regression Surface

- `cmd/bigclawctl`
  CLI parsing, exit-code propagation, output-path handling, and JSON rendering for migration utilities.
- `internal/migration`
  Scorecard and gate semantics, evidence-path resolution, and compatibility with checked-in validation bundle artifacts.
- `docs/e2e-validation.md`
  Operator-facing command paths for continuation evidence refresh.
- `internal/api/validation_bundle_continuation_surface.go`
  Consumer of the generated scorecard/gate artifacts; field compatibility must remain stable.

## Branch / PR Recommendation

- Branch: `codex/big-go-907-validation-reporting-migration`
- PR title: `BIG-GO-907: migrate validation continuation reporting path to Go`
- PR scope:
  Keep the code change limited to the continuation scorecard/gate path, CLI wiring, and migration planning/docs. Do not mix broader bundle-export or live-shadow rewrites into this PR.

## Risks

- Artifact-schema drift:
  `internal/api` and checked-in regression tests expect stable JSON field names. Any follow-up migration must preserve those contracts or update the consuming surfaces in the same PR.
- Path-resolution drift:
  Existing checked-in evidence mixes repo-root and `bigclaw-go`-root relative paths. Future migrations should keep explicit path-resolution tests.
- Partial migration:
  Python remains in the bundle export and live-shadow export paths, so unattended workflows are not yet fully Go-native end-to-end.
- Documentation skew:
  Operator docs can diverge from the active CLI if future migration commands change names or defaults without updating `docs/e2e-validation.md`.
