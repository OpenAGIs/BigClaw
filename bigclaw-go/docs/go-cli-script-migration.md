# Go CLI Script Migration

Issues: `BIG-GO-902`, `BIG-GO-1053`, `BIG-GO-1160`, `BIG-GO-1162`

## Current Go-Only Entrypoints

`bigclaw-go/scripts/e2e/` is now a Python-free operator surface. `BIG-GO-1053`
completed the tranche-2 cleanup by keeping only Go-native
`bigclawctl automation ...` subcommands plus the retained shell wrappers needed
for the bundled live-validation workflow.

| Active entrypoint | Backing command | Purpose |
| --- | --- | --- |
| `go run ./cmd/bigclawctl automation e2e run-task-smoke ...` | `bigclawctl automation e2e run-task-smoke` | Generic submit/poll smoke helper for local, Kubernetes, and Ray paths |
| `go run ./cmd/bigclawctl automation e2e export-validation-bundle ...` | `bigclawctl automation e2e export-validation-bundle` | Export timestamped validation bundles plus latest-summary/index artifacts |
| `go run ./cmd/bigclawctl automation e2e continuation-scorecard ...` | `bigclawctl automation e2e continuation-scorecard` | Regenerate the continuation scorecard from checked-in bundle evidence |
| `go run ./cmd/bigclawctl automation e2e continuation-policy-gate ...` | `bigclawctl automation e2e continuation-policy-gate` | Evaluate continuation readiness from the scorecard surface |
| `go run ./cmd/bigclawctl automation e2e broker-failover-stub-matrix ...` | `bigclawctl automation e2e broker-failover-stub-matrix` | Build the broker failover stub matrix report |
| `go run ./cmd/bigclawctl automation e2e mixed-workload-matrix ...` | `bigclawctl automation e2e mixed-workload-matrix` | Emit the mixed-workload fairness matrix |
| `go run ./cmd/bigclawctl automation e2e cross-process-coordination-surface ...` | `bigclawctl automation e2e cross-process-coordination-surface` | Render the coordination capability surface |
| `go run ./cmd/bigclawctl automation e2e subscriber-takeover-fault-matrix ...` | `bigclawctl automation e2e subscriber-takeover-fault-matrix` | Generate deterministic takeover-fault coverage |
| `go run ./cmd/bigclawctl automation e2e external-store-validation ...` | `bigclawctl automation e2e external-store-validation` | Exercise the external-store replay/takeover validation lane |
| `go run ./cmd/bigclawctl automation e2e multi-node-shared-queue ...` | `bigclawctl automation e2e multi-node-shared-queue` | Generate the live shared-queue companion proof |
| `./scripts/e2e/run_all.sh` | orchestrates Go subcommands plus smoke wrappers | Run local/Kubernetes/Ray live validation and refresh bundle artifacts |
| `./scripts/e2e/kubernetes_smoke.sh` | wraps `automation e2e run-task-smoke` | Run the Kubernetes smoke lane with repo defaults |
| `./scripts/e2e/ray_smoke.sh` | wraps `automation e2e run-task-smoke` | Run the Ray smoke lane with repo defaults |
| `go run ./cmd/bigclawctl automation benchmark soak-local|run-matrix|capacity-certification ...` | `bigclawctl automation benchmark ...` | Benchmark and capacity-certification surfaces |
| `go run ./cmd/bigclawctl automation migration shadow-compare|shadow-matrix|live-shadow-scorecard|export-live-shadow-bundle ...` | `bigclawctl automation migration ...` | Migration shadow comparison and export surfaces |

## BIG-GO-1160 Sweep Coverage

`BIG-GO-1160` validates that the remaining Python candidate paths in this lane
stay retired and that operators keep using the Go-native replacements below.
The current branch baseline is already Python-free for these assets, so the
regression surface focuses on keeping the deletion state sticky.

| Retired sweep area | Supported replacement |
| --- | --- |
| Benchmark soak/matrix/capacity helpers and their Python-side tests | `go run ./cmd/bigclawctl automation benchmark soak-local ...`, `go run ./cmd/bigclawctl automation benchmark run-matrix ...`, `go run ./cmd/bigclawctl automation benchmark capacity-certification ...`, `go test ./cmd/bigclawctl -run TestAutomationBenchmarkCapacityCertificationBuildsReport` |
| E2E broker failover, coordination, bundle export, external-store, workload, shared-queue, smoke, takeover, and continuation sweep candidates | `go run ./cmd/bigclawctl automation e2e broker-failover-stub-matrix ...`, `go run ./cmd/bigclawctl automation e2e cross-process-coordination-surface ...`, `go run ./cmd/bigclawctl automation e2e export-validation-bundle ...`, `go run ./cmd/bigclawctl automation e2e external-store-validation ...`, `go run ./cmd/bigclawctl automation e2e mixed-workload-matrix ...`, `go run ./cmd/bigclawctl automation e2e multi-node-shared-queue ...`, `./scripts/e2e/run_all.sh`, `go run ./cmd/bigclawctl automation e2e run-task-smoke ...`, `go run ./cmd/bigclawctl automation e2e subscriber-takeover-fault-matrix ...`, `go run ./cmd/bigclawctl automation e2e continuation-policy-gate ...`, `go run ./cmd/bigclawctl automation e2e continuation-scorecard ...` |
| Migration shadow compare/matrix/scorecard/export helpers | `go run ./cmd/bigclawctl automation migration export-live-shadow-bundle ...`, `go run ./cmd/bigclawctl automation migration live-shadow-scorecard ...`, `go run ./cmd/bigclawctl automation migration shadow-compare ...`, `go run ./cmd/bigclawctl automation migration shadow-matrix ...` |
| Root create-issues and dev-smoke helpers | `bash scripts/ops/bigclawctl create-issues ...`, `bash scripts/ops/bigclawctl dev-smoke` |

## BIG-GO-1162 Residual Test Sweep Coverage

`BIG-GO-1162` hardens the retired repository-root `tests/*.py` tranche that used
to validate Python-era audit, connector, console, control-plane, planning,
queue, repo, reporting, and migration shadow surfaces. The branch baseline is
already at zero real `.py` files, so this lane keeps the scope on preventing
reintroduction and pinning the supported Go-native replacements that cover the
same operator and report surfaces.

| Retired Python test tranche | Supported replacement |
| --- | --- |
| `tests/test_audit_events.py`, `tests/test_observability.py`, `tests/test_reports.py` | `bigclaw-go/internal/observability/audit_test.go`, `bigclaw-go/internal/observability/recorder_test.go`, `bigclaw-go/internal/reporting/reporting_test.go`, `bigclaw-go/docs/reports/go-control-plane-observability-report.md` |
| `tests/test_connectors.py`, `tests/test_mapping.py`, `tests/test_models.py` | `bigclaw-go/internal/intake/connector_test.go`, `bigclaw-go/internal/intake/mapping_test.go`, `bigclaw-go/internal/workflow/model_test.go` |
| `tests/test_console_ia.py`, `tests/test_design_system.py`, `tests/test_dashboard_run_contract.py`, `tests/test_control_center.py` | `bigclaw-go/internal/consoleia/consoleia_test.go`, `bigclaw-go/internal/designsystem/designsystem_test.go`, `bigclaw-go/internal/product/dashboard_run_contract_test.go`, `bigclaw-go/internal/api/server_test.go` |
| `tests/test_cost_control.py`, `tests/test_execution_contract.py`, `tests/test_execution_flow.py`, `tests/test_orchestration.py`, `tests/test_planning.py`, `tests/test_queue.py` | `bigclaw-go/internal/costcontrol/controller_test.go`, `bigclaw-go/internal/contract/execution_test.go`, `bigclaw-go/internal/workflow/orchestration_test.go`, `bigclaw-go/internal/planning/planning_test.go`, `bigclaw-go/internal/queue/sqlite_queue_test.go` |
| `tests/test_cross_process_coordination_surface.py`, `tests/test_parallel_refill.py`, `tests/test_parallel_validation_bundle.py`, `tests/test_followup_digests.py`, `tests/test_live_shadow_bundle.py`, `tests/test_live_shadow_scorecard.py` | `bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go`, `bigclaw-go/internal/regression/parallel_validation_matrix_docs_test.go`, `bigclaw-go/internal/regression/live_shadow_bundle_surface_test.go`, `bigclaw-go/cmd/bigclawctl/automation_commands_test.go`, `bigclaw-go/docs/reports/parallel-follow-up-index.md` |
| `tests/test_github_sync.py`, `tests/test_governance.py`, `tests/test_issue_archive.py`, `tests/test_operations.py`, `tests/test_pilot.py` | `bigclaw-go/internal/githubsync/sync_test.go`, `bigclaw-go/internal/governance/freeze_test.go`, `bigclaw-go/internal/issuearchive/archive_test.go`, `bigclaw-go/docs/reports/v2-phase1-operations-foundation-report.md`, `bigclaw-go/internal/pilot/rollout_test.go` |
| `tests/test_repo_board.py`, `tests/test_repo_collaboration.py`, `tests/test_repo_gateway.py`, `tests/test_repo_governance.py`, `tests/test_repo_links.py`, `tests/test_repo_registry.py`, `tests/test_repo_rollout.py`, `tests/test_repo_triage.py` | `bigclaw-go/internal/repo/repo_surfaces_test.go`, `bigclaw-go/internal/collaboration/thread_test.go`, `bigclaw-go/internal/repo/gateway.go`, `bigclaw-go/internal/repo/governance.go`, `bigclaw-go/internal/repo/links.go`, `bigclaw-go/internal/repo/registry.go`, `bigclaw-go/internal/product/clawhost_rollout_test.go`, `bigclaw-go/internal/triage/repo_test.go` |

## Validation Commands

```bash
cd bigclaw-go
go test ./cmd/bigclawctl/...
go run ./cmd/bigclawctl automation --help
go run ./cmd/bigclawctl automation e2e run-task-smoke --help
go run ./cmd/bigclawctl automation e2e export-validation-bundle --help
go run ./cmd/bigclawctl automation e2e continuation-scorecard --help
go run ./cmd/bigclawctl automation e2e continuation-policy-gate --help
go run ./cmd/bigclawctl automation e2e broker-failover-stub-matrix --help
go run ./cmd/bigclawctl automation e2e mixed-workload-matrix --help
go run ./cmd/bigclawctl automation e2e cross-process-coordination-surface --help
go run ./cmd/bigclawctl automation e2e subscriber-takeover-fault-matrix --help
go run ./cmd/bigclawctl automation e2e external-store-validation --help
go run ./cmd/bigclawctl automation e2e multi-node-shared-queue --help
go run ./cmd/bigclawctl automation benchmark soak-local --help
go run ./cmd/bigclawctl automation benchmark run-matrix --help
go run ./cmd/bigclawctl automation benchmark capacity-certification --help
go run ./cmd/bigclawctl automation migration shadow-compare --help
go run ./cmd/bigclawctl automation migration shadow-matrix --help
go run ./cmd/bigclawctl automation migration live-shadow-scorecard --help
go run ./cmd/bigclawctl automation migration export-live-shadow-bundle --help
```

## Regression Surface

- CLI parsing and root help for `bigclawctl`
- HTTP polling against `/healthz`, `/tasks/:id`, and `/events`
- Temporary `bigclawd` autostart state wiring for smoke and soak commands
- Report serialization compatibility for JSON consumers that previously read the Python script output
## Compatibility Layer Plan

- Keep new behavior in Go-native entrypoints and do not reintroduce Python helpers under `bigclaw-go/scripts/e2e/`.
- Preserve the retained shell wrappers only where they add operator convenience over direct `bigclawctl automation ...` invocation.
- Continue the remaining non-e2e script migrations in follow-up batches without expanding the e2e compatibility layer again.

## Branch And PR Suggestion

- Branch: `feat/BIG-GO-902-go-cli-script-migration`
- PR title: `feat: migrate first Python automation scripts to bigclawctl`

## Risks

- `soak-local` now uses Go worker concurrency; very large counts may stress a single local HTTP backend differently than the old Python thread pool.
- `run-task-smoke --autostart` and `soak-local --autostart` still rely on ephemeral port reservation before `bigclawd` binds, so local port races remain possible.
- The shell wrappers in `scripts/e2e/` remain convenience layers; changes to flag defaults must stay aligned with the underlying Go subcommands.
