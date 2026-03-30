# BIG-GO-983 Workpad

## Scope

Targeted `src/bigclaw/**` core-module cleanup batch for files that already have a Go-owned mainline replacement and no remaining in-repo Python imports beyond legacy package exports.

Candidate batch:

- `src/bigclaw/mapping.py`
- `src/bigclaw/issue_archive.py`
- `src/bigclaw/parallel_refill.py`
- `src/bigclaw/pilot.py`
- `src/bigclaw/workspace_bootstrap_cli.py`
- `src/bigclaw/cost_control.py`
- `src/bigclaw/roadmap.py`
- `src/bigclaw/connectors.py`
- `src/bigclaw/github_sync.py`
- `src/bigclaw/workspace_bootstrap_validation.py`
- `src/bigclaw/repo_governance.py`
- `src/bigclaw/repo_board.py`
- `src/bigclaw/repo_gateway.py`
- `src/bigclaw/repo_registry.py`
- `src/bigclaw/repo_triage.py`
- `src/bigclaw/dsl.py`
- `src/bigclaw/dashboard_run_contract.py`
- `src/bigclaw/saved_views.py`
- `src/bigclaw/event_bus.py`
- `src/bigclaw/execution_contract.py`

Planned retainers for this lane:

- none

Current repository Python file count before this lane: `116`
Current `src/bigclaw/**` Python file count before this lane: `45`

## Plan

1. Confirm each candidate file is either unreferenced in the remaining Python tree or only reachable through stale package exports.
2. Validate the existing Go replacement paths for intake mapping, issue archive, refill/bootstrap tooling, and pilot reporting.
3. Delete the selected Python files and remove stale imports/exports from `src/bigclaw/__init__.py`.
4. Run targeted Go validation for the replacement packages and recount Python files.
5. Record per-file keep/delete rationale, exact commands, and before/after counts.

## Acceptance

- Produce the exact `BIG-GO-983` batch file list and disposition.
- Reduce Python files under `src/bigclaw/**` by removing the safely migrated subset.
- Keep changes scoped to this core-module cleanup batch.
- Report the repository-wide and `src/bigclaw/**` Python file-count impact.
- Record exact validation commands and results.

## Validation

- `cd bigclaw-go && go test ./internal/intake ./internal/issuearchive ./internal/refill ./internal/pilot ./cmd/bigclawctl`
- `python3 -m compileall src/bigclaw/__init__.py`
- `rg --files src/bigclaw -g '*.py' | wc -l`
- `rg --files -g '*.py' | wc -l`
- `git status --short`

## Results

### File Disposition

- `src/bigclaw/mapping.py`
  - Deleted.
  - Reason: no remaining in-repo Python imports; parity matrix maps the full surface to `bigclaw-go/internal/intake/mapping.go`, and Go intake tests already own the active mapping contract.
- `src/bigclaw/issue_archive.py`
  - Deleted.
  - Reason: no remaining in-repo Python imports; the equivalent archive/audit/report surface exists in `bigclaw-go/internal/issuearchive/archive.go` with package tests.
- `src/bigclaw/parallel_refill.py`
  - Deleted.
  - Reason: no remaining in-repo Python imports; cutover docs assign refill ownership to `bigclaw-go/internal/refill/*`, and the queue behavior is covered by Go refill tests plus `cmd/bigclawctl`.
- `src/bigclaw/pilot.py`
  - Deleted.
  - Reason: no remaining in-repo Python imports; the same implementation-report surface exists in `bigclaw-go/internal/pilot/report.go` with Go tests.
- `src/bigclaw/workspace_bootstrap_cli.py`
  - Deleted.
  - Reason: no remaining in-repo Python imports; cutover docs assign bootstrap CLI ownership to `bigclaw-go/cmd/bigclawctl` and `bigclaw-go/internal/bootstrap/*`, which are already covered by Go tests.
- `src/bigclaw/cost_control.py`
  - Deleted.
  - Reason: no remaining in-repo Python imports; equivalent budget-evaluation behavior exists in `bigclaw-go/internal/costcontrol/controller.go` with package tests.
- `src/bigclaw/roadmap.py`
  - Deleted.
  - Reason: no remaining in-repo Python imports; Go regression coverage in `bigclaw-go/internal/regression/roadmap_contract_test.go` now owns the canonical execution-pack roadmap contract.
- `src/bigclaw/connectors.py`
  - Deleted.
  - Reason: no remaining in-repo Python imports; parity matrix maps `SourceIssue` and connector stubs to `bigclaw-go/internal/intake/types.go` and `bigclaw-go/internal/intake/connector.go`, both covered by Go intake tests.
- `src/bigclaw/github_sync.py`
  - Deleted.
  - Reason: only legacy Python tests referenced it; cutover docs assign this surface to `bigclaw-go/internal/githubsync/*` and `bigclaw-go/cmd/bigclawctl`, both already covered by Go tests.
- `src/bigclaw/workspace_bootstrap_validation.py`
  - Deleted.
  - Reason: only legacy Python tests referenced it; cutover docs assign validation/report generation to `bigclaw-go/internal/bootstrap/*` and `bigclaw-go/cmd/bigclawctl workspace validate`.
- `src/bigclaw/repo_governance.py`
  - Deleted.
  - Reason: only legacy Python tests referenced it; cutover docs explicitly map it to `bigclaw-go/internal/repo/governance.go`, with Go tests covering the permission matrix and audit-field contract.
- `src/bigclaw/repo_board.py`
  - Deleted.
  - Reason: only legacy Python tests referenced it; the repo discussion surface now lives under `bigclaw-go/internal/repo/board.go` and `repo_surfaces_test.go`.
- `src/bigclaw/repo_gateway.py`
  - Deleted.
  - Reason: only legacy Python tests referenced it; the normalized gateway payload/error surface now lives under `bigclaw-go/internal/repo/gateway.go` and `repo_surfaces_test.go`.
- `src/bigclaw/repo_registry.py`
  - Deleted.
  - Reason: only legacy Python tests referenced it; the registry/channel/agent surface now lives under `bigclaw-go/internal/repo/registry.go` and `repo_surfaces_test.go`.
- `src/bigclaw/repo_triage.py`
  - Deleted.
  - Reason: only legacy Python tests referenced it; the lineage-aware triage surface now lives under `bigclaw-go/internal/repo/triage.go` and `repo_surfaces_test.go`.
- `src/bigclaw/dsl.py`
  - Deleted.
  - Reason: only legacy Python tests referenced it; parity docs map the workflow-definition contract to `bigclaw-go/internal/workflow/definition.go`, covered by Go workflow tests.
- `src/bigclaw/dashboard_run_contract.py`
  - Deleted.
  - Reason: only legacy Python tests referenced it; the active dashboard/run contract now lives under `bigclaw-go/internal/product/dashboard_run_contract.go` with Go tests.
- `src/bigclaw/saved_views.py`
  - Deleted.
  - Reason: only legacy Python tests referenced it; the active saved-view/digest surface now lives under `bigclaw-go/internal/product/saved_views.go` with Go tests.
- `src/bigclaw/event_bus.py`
  - Deleted.
  - Reason: only legacy Python tests referenced it; the active event bus surface now lives under `bigclaw-go/internal/events/bus.go` with Go tests.
- `src/bigclaw/execution_contract.py`
  - Deleted.
  - Reason: only legacy Python tests referenced it; cutover docs explicitly assign the execution contract to `bigclaw-go/internal/contract/execution.go`, with Go tests covering the contract audit, role matrix, and operations API draft.
- `src/bigclaw/__init__.py`
  - Updated.
  - Reason: removed stale package-level imports/exports for deleted modules so `import bigclaw` no longer hard-fails on removed files.
- `tests/test_github_sync.py`
  - Deleted.
  - Reason: legacy Python-only test for a module removed in this lane; equivalent behavior is validated by Go package tests.
- `tests/test_workspace_bootstrap.py`
  - Updated.
  - Reason: removed the helper-specific validation-report test that only covered the deleted Python compatibility wrapper; remaining tests still cover `workspace_bootstrap.py`.
- `tests/test_repo_governance.py`
  - Deleted.
  - Reason: legacy Python-only test for a module removed in this lane; equivalent behavior is covered by `bigclaw-go/internal/repo/governance_test.go`.
- `tests/test_repo_board.py`
  - Deleted.
  - Reason: legacy Python-only test for a module removed in this lane; equivalent behavior is covered by `bigclaw-go/internal/repo/repo_surfaces_test.go`.
- `tests/test_repo_gateway.py`
  - Deleted.
  - Reason: legacy Python-only test for a module removed in this lane; equivalent behavior is covered by `bigclaw-go/internal/repo/repo_surfaces_test.go`.
- `tests/test_repo_registry.py`
  - Deleted.
  - Reason: legacy Python-only test for a module removed in this lane; equivalent behavior is covered by `bigclaw-go/internal/repo/repo_surfaces_test.go`.
- `tests/test_repo_triage.py`
  - Deleted.
  - Reason: legacy Python-only test for a module removed in this lane; equivalent behavior is covered by `bigclaw-go/internal/repo/repo_surfaces_test.go`.
- `tests/test_repo_collaboration.py`
  - Deleted.
  - Reason: it existed only to exercise the deleted Python repo-board compatibility surface.
- `tests/test_dsl.py`
  - Deleted.
  - Reason: legacy Python-only test for a module removed in this lane; equivalent behavior is covered by `bigclaw-go/internal/workflow/definition_test.go`.
- `tests/test_dashboard_run_contract.py`
  - Deleted.
  - Reason: legacy Python-only test for a module removed in this lane; equivalent behavior is covered by `bigclaw-go/internal/product/dashboard_run_contract_test.go`.
- `tests/test_saved_views.py`
  - Deleted.
  - Reason: legacy Python-only test for a module removed in this lane; equivalent behavior is covered by `bigclaw-go/internal/product/saved_views_test.go`.
- `tests/test_event_bus.py`
  - Deleted.
  - Reason: legacy Python-only test for a module removed in this lane; equivalent behavior is covered by `bigclaw-go/internal/events/bus_test.go`.
- `tests/test_execution_contract.py`
  - Deleted.
  - Reason: legacy Python-only test for a module removed in this lane; equivalent behavior is covered by `bigclaw-go/internal/contract/execution_test.go`.

### Python File Count Impact

- Repository Python files before: `116`
- Repository Python files after: `84`
- `src/bigclaw/**` Python files before: `45`
- `src/bigclaw/**` Python files after: `25`
- Net reduction: `32`

### Validation Record

- `cd bigclaw-go && go test ./internal/intake ./internal/issuearchive ./internal/refill ./internal/pilot ./cmd/bigclawctl`
  - Result: `ok  	bigclaw-go/internal/intake	1.125s`
  - Result: `ok  	bigclaw-go/internal/issuearchive	1.556s`
  - Result: `ok  	bigclaw-go/internal/refill	4.718s`
  - Result: `ok  	bigclaw-go/internal/pilot	1.840s`
  - Result: `ok  	bigclaw-go/cmd/bigclawctl	3.818s`
- `python3 -m compileall src/bigclaw/__init__.py`
  - Result: `Compiling 'src/bigclaw/__init__.py'...`
- `cd bigclaw-go && go test ./internal/costcontrol ./internal/regression`
  - Result: `ok  	bigclaw-go/internal/costcontrol	1.088s`
  - Result: `ok  	bigclaw-go/internal/regression	1.332s`
- `cd bigclaw-go && go test ./internal/intake`
  - Result: `ok  	bigclaw-go/internal/intake	(cached)`
- `cd bigclaw-go && go test ./internal/githubsync ./internal/bootstrap ./cmd/bigclawctl`
  - Result: `ok  	bigclaw-go/internal/githubsync	(cached)`
  - Result: `ok  	bigclaw-go/internal/bootstrap	(cached)`
  - Result: `ok  	bigclaw-go/cmd/bigclawctl	(cached)`
- `cd bigclaw-go && go test ./internal/repo`
  - Result: `ok  	bigclaw-go/internal/repo	1.150s`
- `cd bigclaw-go && go test ./internal/workflow ./internal/product ./internal/events`
  - Result: `ok  	bigclaw-go/internal/workflow	1.129s`
  - Result: `ok  	bigclaw-go/internal/product	1.497s`
  - Result: `ok  	bigclaw-go/internal/events	2.062s`
- `cd bigclaw-go && go test ./internal/contract`
  - Result: `ok  	bigclaw-go/internal/contract	1.188s`
- `git status --short`
  - Result: later cleanup waves removed repo compatibility modules, workflow/product/events compatibility modules, and their Python-only tests.
