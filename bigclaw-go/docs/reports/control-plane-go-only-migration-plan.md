# Control Plane Go-only Migration Slices

This report records the `BIG-GO-904` control-plane inventory for remaining non-Go dependencies, the first executable migration slice, and the follow-on cuts needed to remove Python from the control-plane review path.

## Scope

- In scope: `bigclaw-go` control-plane APIs, migration review surfaces, compatibility shims reached through `bigclawctl`, and repo-native migration evidence generation.
- Out of scope for this slice: root-level legacy Python runtime replacement, full `scripts/e2e/*.py` harness migration, and service-plane fault-injection rewrite.

## Non-Go dependency inventory

| Surface | Current dependency | Class | Why it still exists | Proposed Go-only slice |
| --- | --- | --- | --- | --- |
| `scripts/migration/live_shadow_scorecard.py` | `python3` | Review artifact generation | Produces `live-shadow-mirror-scorecard.json`, which is consumed by `GET /debug/status` and `GET /v2/control-center` reviewer surfaces. | Replace with `bigclawctl migration live-shadow-scorecard`. Implemented in this issue. |
| `scripts/migration/export_live_shadow_bundle.py` | `python3` | Review artifact packaging | Copies compare/matrix/scorecard/rollback artifacts into `docs/reports/live-shadow-runs/<run-id>/` and refreshes bundle indexes. | Port bundle/index exporter to Go after scorecard cutover to keep one toolchain for migration review artifacts. |
| `scripts/migration/shadow_compare.py` | `python3` | Control-plane comparison harness | Submits one fixture payload to primary/shadow endpoints and compares terminal states/events. | Rebuild as Go HTTP harness reusing the same task/event schema already consumed by `bigclawd`. |
| `scripts/migration/shadow_matrix.py` | `python3` | Control-plane comparison harness | Runs the fixture/corpus matrix used by migration readiness reports. | Port alongside `shadow_compare.py` so the matrix shares one Go diff engine. |
| `bigclawctl legacy-python compile-check` -> `src/bigclaw/*.py` | `python3` + frozen legacy Python files | Compatibility validation | Protects the frozen Python shim while the repo still ships legacy entrypoints. | Keep as compatibility-only gate until root runtime retirement is scheduled; then remove command and frozen file list together. |
| `docs/e2e-validation.md` + `scripts/e2e/*.py` | `python3` | Service-plane validation harness | Multi-node/local/broker drills are still orchestrated from Python wrappers. | Migrate after control-plane review path is Go-only, because these scripts are broader and touch multi-process orchestration. |

## First batch implemented here

1. Cut the live shadow scorecard generator over to Go with `go run ./cmd/bigclawctl migration live-shadow-scorecard`.
2. Update migration docs and checked-in scorecard metadata so the reviewer path no longer depends on `scripts/migration/live_shadow_scorecard.py`.
3. Keep the exporter, compare harness, matrix harness, and legacy compile-check unchanged to avoid mixing control-plane artifact migration with e2e/runtime retirement.

## Execution plan

1. Land Go-native scorecard generation and keep the JSON schema stable for `internal/api/live_shadow_surface.go`.
2. Port `export_live_shadow_bundle.py` next so bundle refresh and scorecard generation share the same Go CLI.
3. Port `shadow_compare.py` and `shadow_matrix.py` together, then switch `docs/migration-shadow.md` and migration readiness docs to Go-only commands.
4. After the repo-native review toolchain is Go-only, separate the legacy Python shim retirement from the broader `scripts/e2e/*.py` service-plane harness migration.

## Validation commands

- `cd bigclaw-go && go test ./internal/migration ./cmd/bigclawctl ./internal/regression`
- `cd bigclaw-go && go run ./cmd/bigclawctl migration live-shadow-scorecard --repo .`
- `cd bigclaw-go && rg -n "live-shadow-scorecard|control-plane-go-only" docs cmd internal`

## Regression surface

- `internal/api/live_shadow_surface.go` must continue decoding `docs/reports/live-shadow-mirror-scorecard.json` without schema drift.
- `docs/migration-shadow.md`, `docs/reports/migration-readiness-report.md`, and bundled reviewer docs must keep commands and caveats aligned.
- `bigclawctl legacy-python` must remain untouched in behavior because other repo areas still use the frozen Python compatibility gate.

## Branch and PR suggestion

- Branch: `codex/BIG-GO-904-control-plane-go-only-slice`
- PR title: `BIG-GO-904: cut live shadow scorecard generation to Go`
- PR body focus:
  - Inventory of remaining non-Go control-plane dependencies
  - This slice's Go-native scorecard generator
  - Follow-on slices for exporter/compare/matrix migration
  - Explicit non-goals for legacy runtime and e2e harnesses

## Risks

- The checked-in scorecard schema is runtime-facing; field drift would break control-center and debug-status reviewer payloads.
- Docs can easily drift from the generator command because bundle READMEs duplicate the same workflow text.
- Porting only the scorecard leaves Python in the exporter and compare/matrix harness, so reviewers still need two toolchains until follow-on slices land.
