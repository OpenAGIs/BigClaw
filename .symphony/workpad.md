# BIG-GO-1015 Workpad

## Scope

Target tranche 3 and follow-on tranche 4 of the remaining `src/bigclaw/**`
repository-surface Python helpers that already have repo-native Go replacements
and no longer need to exist as active Python modules.

Batch file list:

- `src/bigclaw/repo_board.py`
- `src/bigclaw/repo_commits.py`
- `src/bigclaw/repo_gateway.py`
- `src/bigclaw/repo_governance.py`
- `src/bigclaw/repo_registry.py`
- `src/bigclaw/repo_triage.py`
- `src/bigclaw/github_sync.py`
- `src/bigclaw/parallel_refill.py`
- `src/bigclaw/workspace_bootstrap_cli.py`
- `src/bigclaw/cost_control.py`
- `src/bigclaw/issue_archive.py`
- `src/bigclaw/roadmap.py`
- `src/bigclaw/validation_policy.py`
- `src/bigclaw/dashboard_run_contract.py`
- `src/bigclaw/memory.py`
- `src/bigclaw/connectors.py`
- `src/bigclaw/mapping.py`
- `src/bigclaw/pilot.py`
- `src/bigclaw/workspace_bootstrap.py`
- `src/bigclaw/workspace_bootstrap_validation.py`
- `src/bigclaw/__main__.py`
- `bigclaw-go/internal/legacyshim/compilecheck.go`
- `bigclaw-go/internal/legacyshim/compilecheck_test.go`
- `bigclaw-go/cmd/bigclawctl/main_test.go`
- `bigclaw-go/internal/regression/deprecation_contract_test.go`
- `bigclaw-go/docs/reports/legacy-mainline-compatibility-manifest.json`
- `src/bigclaw/__init__.py`
- `tests/test_repo_board.py`
- `tests/test_repo_gateway.py`
- `tests/test_repo_governance.py`
- `tests/test_repo_registry.py`
- `tests/test_repo_triage.py`
- `tests/test_repo_collaboration.py`
- `tests/test_github_sync.py`

Context at start of lane:

- `src/bigclaw` Python files: `45`
- `bigclaw-go` Go files: `267`
- Root `pyproject.toml`: absent
- Root `setup.py`: absent

Keep-out files for this lane:

- `src/bigclaw/repo_links.py`
- `src/bigclaw/repo_plane.py`
- `src/bigclaw/observability.py`

Reason:

- `observability.py` still imports `repo_links.bind_run_commits` and
  `repo_plane.RunCommitLink`, so deleting those modules in this lane would
  broaden scope beyond the safe tranche.

Follow-on refactor now queued:

- Move `RunCommitLink`, `RunCommitBinding`, and run-commit role validation into
  `src/bigclaw/observability.py`.
- Repoint the two remaining tests at the active `observability` surface.
- Delete `src/bigclaw/repo_links.py` and `src/bigclaw/repo_plane.py` once
  their last in-repo references are gone.
- Inline the tiny `src/bigclaw/deprecation.py` warning helpers into
  `src/bigclaw/runtime.py` and repoint `scripts/dev_smoke.py` if the helper
  remains isolated after the repo-link cleanup.
- Inline the small `src/bigclaw/risk.py` scorer surface into
  `src/bigclaw/runtime.py` if its remaining consumers stay limited to runtime,
  tests, and package exports.
- Inline the small `src/bigclaw/event_bus.py` surface into
  `src/bigclaw/observability.py` if its remaining consumers stay limited to
  observability, tests, and package exports.
- Inline the small `src/bigclaw/governance.py` surface into
  `src/bigclaw/planning.py` if its remaining consumers stay limited to
  planning, tests, and package exports.
- Inline the isolated `src/bigclaw/execution_contract.py` surface into
  `src/bigclaw/operations.py` if it remains limited to tests and package
  exports, since its concrete builder already targets the operations API.

## Plan

1. Remove the six repo-surface Python modules that already map to checked-in Go
   implementations under `bigclaw-go/internal/repo` and
   `bigclaw-go/internal/triage`.
2. Remove the migration-era Python operator helpers already superseded by
   `bigclawctl` and `bigclaw-go/internal/{bootstrap,githubsync,refill}`.
3. Remove isolated Python compatibility/data modules with direct Go ownership
   and no remaining in-repo consumers.
4. Remove isolated Python policy/helpers that are now only covered by stale
   package exports or dedicated Python-only tests.
5. Remove isolated Python contract modules that already have Go-owned contract
   implementations and only retain Python-only tests/package exports.
6. Remove isolated Python-only helpers that have no remaining in-repo runtime
   consumers and are retained only by dedicated tests.
7. Remove isolated Python intake/mapping modules that already have documented
   Go parity ownership and now only survive via stale package exports.
8. Remove isolated Python modules that have no remaining package/runtime users
   and already have repo-native Go replacements.
9. Remove isolated Python bootstrap helpers whose ownership is already moved to
   the Go bootstrap toolchain and which only survive via dedicated tests.
10. Remove stale Python entrypoint residue and reconcile Go-side compatibility
    evidence/tests that still claim the deleted entrypoints exist.
11. Remove Python tests that only exercised deleted Python-only modules.
12. Update any remaining Python tests or exports that referenced removed modules so
    the suite remains coherent after deletion.
13. Run targeted validation for remaining observability, repo-link, and
    workspace bootstrap surfaces,
    plus inventory counts and diff hygiene.
14. Commit and push the scoped lane branch for `BIG-GO-1015`.

## Acceptance

- Directly reduce repository-resident Python assets under `src/bigclaw/**`.
- Keep changes scoped to the tranche-3 repo helper slice only.
- Report exact impact on `py files`, `go files`, `pyproject.toml`, and
  `setup.py`.
- Record exact validation commands and results.
- End with committed and pushed repository changes; do not substitute tracker
  state for repo results.

## Validation

- `find src/bigclaw -type f -name '*.py' | wc -l`
- `find bigclaw-go -type f -name '*.go' | wc -l`
- `test -f pyproject.toml && echo present || echo absent`
- `test -f setup.py && echo present || echo absent`
- `PYTHONPATH=src python3 -m pytest tests/test_observability.py tests/test_repo_links.py tests/test_repo_rollout.py`
- `PYTHONPATH=src python3 -m pytest tests/test_workspace_bootstrap.py`
- `PYTHONPATH=src python3 -m pytest tests/test_workspace_bootstrap.py tests/test_observability.py tests/test_repo_links.py tests/test_repo_rollout.py tests/test_repo_collaboration.py`
- `PYTHONPATH=src python3 -m pytest tests/test_validation_policy.py`
- `rg -n "github_sync|parallel_refill|workspace_bootstrap_cli" src tests || true`
- `rg -n "roadmap|validation_policy" src tests || true`
- `python3 - <<'PY'` to import `bigclaw` after trimming stale package exports
- `git diff --check`
- `git status --short`
- `git log -1 --stat`

## Results

### File Disposition

- `src/bigclaw/repo_board.py`
  - Deleted.
  - Reason: repo discussion-board helper already exists in
    `bigclaw-go/internal/repo/board.go` and had no remaining production Python
    callers.
- `src/bigclaw/repo_commits.py`
  - Deleted.
  - Reason: repo commit and lineage structs already exist in
    `bigclaw-go/internal/repo/commits.go`; remaining Python usage was only the
    deleted gateway test.
- `src/bigclaw/repo_gateway.py`
  - Deleted.
  - Reason: gateway client contract and normalization logic already exist in
    `bigclaw-go/internal/repo/gateway.go` and had no remaining Python imports.
- `src/bigclaw/repo_governance.py`
  - Deleted.
  - Reason: repo permission contract already exists in
    `bigclaw-go/internal/repo/governance.go` and had no remaining production
    Python imports.
- `src/bigclaw/repo_registry.py`
  - Deleted.
  - Reason: repo registry logic already exists in
    `bigclaw-go/internal/repo/registry.go` and had no remaining Python imports.
- `src/bigclaw/repo_triage.py`
  - Deleted.
  - Reason: repo triage recommendation logic already exists in
    `bigclaw-go/internal/repo/triage.go` and had no remaining Python imports.
- `src/bigclaw/github_sync.py`
  - Deleted.
  - Reason: GitHub sync install / inspect / push guarantees are already owned by
    `bigclaw-go/internal/githubsync/*` and `scripts/ops/bigclawctl`; remaining
    Python usage was only the deleted dedicated test.
- `src/bigclaw/parallel_refill.py`
  - Deleted.
  - Reason: refill queue selection is already owned by
    `bigclaw-go/internal/refill/*` and the operator path is already routed
    through `bigclawctl refill`.
- `src/bigclaw/workspace_bootstrap_cli.py`
  - Deleted.
  - Reason: the CLI wrapper was migration-only glue for the already-retained
    library code in `workspace_bootstrap.py`; the active operator path is
    `bigclawctl workspace bootstrap`.
- `src/bigclaw/cost_control.py`
  - Deleted.
  - Reason: budget-control logic already exists in
    `bigclaw-go/internal/costcontrol/controller.go` and had no remaining Python
    imports or package exports.
- `src/bigclaw/issue_archive.py`
  - Deleted.
  - Reason: issue archive serialization/audit/reporting already exists in
    `bigclaw-go/internal/issuearchive/archive.go` and had no remaining in-repo
    Python consumers.
- `src/bigclaw/roadmap.py`
  - Deleted.
  - Reason: execution-pack roadmap ownership and uniqueness checks are already
    covered by docs plus `bigclaw-go/internal/regression/roadmap_contract_test.go`,
    and the Python module had no remaining in-repo consumers beyond stale
    package exports.
- `src/bigclaw/validation_policy.py`
  - Deleted.
  - Reason: the module was isolated to its dedicated Python-only test and had
    no remaining in-repo runtime consumers.
- `src/bigclaw/dashboard_run_contract.py`
  - Deleted.
  - Reason: dashboard/run contract ownership already exists in
    `bigclaw-go/internal/product/dashboard_run_contract.go`, and the Python
    module had no remaining in-repo consumers beyond a dedicated test and stale
    package exports.
- `src/bigclaw/memory.py`
  - Deleted.
  - Reason: the helper had no remaining in-repo runtime consumers and was only
    retained by its dedicated Python-only test.
- `src/bigclaw/connectors.py`
  - Deleted.
  - Reason: intake connector ownership is already documented and implemented in
    `bigclaw-go/internal/intake/{types,connector}.go`, and the Python module
    had no remaining in-repo consumers beyond stale package exports.
- `src/bigclaw/mapping.py`
  - Deleted.
  - Reason: source-issue mapping ownership is already documented and implemented
    in `bigclaw-go/internal/intake/mapping.go`, and the Python module had no
    remaining in-repo consumers beyond stale package exports.
- `src/bigclaw/pilot.py`
  - Deleted.
  - Reason: the module had no remaining Python imports or package exports and
    already had a repo-native Go replacement in `bigclaw-go/internal/pilot`.
- `src/bigclaw/workspace_bootstrap.py`
  - Deleted.
  - Reason: bootstrap ownership is already moved to
    `bigclaw-go/internal/bootstrap/*` and the Python module only remained
    through a dedicated Python-only test plus the now-removed validation helper.
- `src/bigclaw/workspace_bootstrap_validation.py`
  - Deleted.
  - Reason: validation ownership is already moved to the Go bootstrap toolchain
    and the Python helper only remained through the deleted bootstrap test.
- `src/bigclaw/__main__.py`
  - Deleted.
  - Reason: the legacy Python entrypoint is no longer needed now that Go-first
    operator/tooling ownership is established, and the remaining repo evidence
    was updated to stop claiming that the deleted entrypoint still exists.
- `bigclaw-go/internal/legacyshim/compilecheck.go`
  - Replaced.
  - Reason: removed deleted Python entrypoints from the frozen compile-check
    list so the Go compatibility check matches repository reality.
- `bigclaw-go/internal/legacyshim/compilecheck_test.go`
  - Replaced.
  - Reason: updated the frozen compile-check expectations to match the retained
    shim file set.
- `bigclaw-go/cmd/bigclawctl/main_test.go`
  - Replaced.
  - Reason: updated the legacy-python JSON-output fixture to match the retained
    shim file list after removing stale Python entrypoints.
- `bigclaw-go/internal/regression/deprecation_contract_test.go`
  - Replaced.
  - Reason: removed assertions for deleted `python -m bigclaw` entrypoint
    warnings from the compatibility manifest contract.
- `bigclaw-go/docs/reports/legacy-mainline-compatibility-manifest.json`
  - Replaced.
  - Reason: removed runtime/service warning entries that referred to the
    deleted Python `__main__` entrypoint.
- `src/bigclaw/design_system.py`
  - Deleted.
  - Reason: console design-system ownership is already moved to
    `bigclaw-go/internal/product/console.go`, and the Python module only
    remained through dedicated tests and stale package exports.
- `src/bigclaw/console_ia.py`
  - Deleted.
  - Reason: console IA ownership is already moved to
    `bigclaw-go/internal/product/console.go`, and the Python module only
    remained through dedicated tests and stale package exports.
- `src/bigclaw/saved_views.py`
  - Deleted.
  - Reason: saved-view ownership is already moved to
    `bigclaw-go/internal/product/saved_views.go`, and the Python module only
    remained through dedicated tests and stale package exports.
- `src/bigclaw/dsl.py`
  - Deleted.
  - Reason: workflow-definition ownership is already moved to
    `bigclaw-go/internal/workflow/definition.go`, and the Python module only
    remained through dedicated tests and stale package exports.
- `src/bigclaw/ui_review.py`
  - Deleted.
  - Reason: reviewer-facing console/UI review surfaces are already covered by
    Go-side console ownership in the cutover plan, and the Python module only
    remained through dedicated tests and stale package exports.
- `src/bigclaw/__init__.py`
  - Replaced.
  - Reason: removed the stale `issue_archive`, `roadmap`,
    `dashboard_run_contract`, `connectors`, `mapping`, `dsl`,
    `design_system`, `console_ia`, `saved_views`, and `ui_review` re-export
    blocks after deleting the underlying Python modules.
- `tests/test_repo_board.py`
  - Deleted.
  - Reason: exercised deleted Python-only board helper.
- `tests/test_repo_gateway.py`
  - Deleted.
  - Reason: exercised deleted Python-only gateway helper.
- `tests/test_repo_governance.py`
  - Deleted.
  - Reason: exercised deleted Python-only governance helper.
- `tests/test_repo_registry.py`
  - Deleted.
  - Reason: exercised deleted Python-only registry helper.
- `tests/test_repo_triage.py`
  - Deleted.
  - Reason: exercised deleted Python-only triage helper.
- `tests/test_github_sync.py`
  - Deleted.
  - Reason: exercised deleted Python-only GitHub sync helper.
- `tests/test_validation_policy.py`
  - Deleted.
  - Reason: exercised deleted Python-only validation policy helper.
- `tests/test_dashboard_run_contract.py`
  - Deleted.
  - Reason: exercised deleted Python-only dashboard/run contract helper.
- `tests/test_memory.py`
  - Deleted.
  - Reason: exercised deleted Python-only memory helper.
- `tests/test_design_system.py`
  - Deleted.
  - Reason: exercised deleted Python-only design-system helper.
- `tests/test_console_ia.py`
  - Deleted.
  - Reason: exercised deleted Python-only console IA helper.
- `tests/test_saved_views.py`
  - Deleted.
  - Reason: exercised deleted Python-only saved-view helper.
- `tests/test_dsl.py`
  - Deleted.
  - Reason: exercised deleted Python-only workflow-definition helper.
- `tests/test_ui_review.py`
  - Deleted.
  - Reason: exercised deleted Python-only UI review helper.
- `tests/test_workspace_bootstrap.py`
  - Deleted.
  - Reason: exercised deleted Python-only bootstrap helpers.
- `tests/test_repo_collaboration.py`
  - Replaced.
  - Reason: preserved the collaboration merge assertion while removing the last
    dependency on the deleted `RepoDiscussionBoard` helper.

### Inventory Impact

- `src/bigclaw` Python files before: `45`
- `src/bigclaw` Python files after tranche 3: `39`
- `src/bigclaw` Python files after tranche 4: `36`
- `src/bigclaw` Python files after tranche 5: `34`
- `src/bigclaw` Python files after tranche 6: `32`
- `src/bigclaw` Python files after tranche 7: `31`
- `src/bigclaw` Python files after tranche 8: `30`
- `src/bigclaw` Python files after tranche 9: `28`
- `src/bigclaw` Python files after tranche 10: `27`
- `src/bigclaw` Python files after tranche 11: `22`
- `src/bigclaw` Python files after tranche 12: `20`
- `src/bigclaw` Python files after tranche 13: `19`
- Net `src/bigclaw` reduction: `26`
- Repository-wide Python files before: `108`
- Repository-wide Python files after tranche 3: `97`
- Repository-wide Python files after tranche 4: `93`
- Repository-wide Python files after tranche 5: `91`
- Repository-wide Python files after tranche 6: `88`
- Repository-wide Python files after tranche 7: `86`
- Repository-wide Python files after tranche 8: `84`
- Repository-wide Python files after tranche 9: `82`
- Repository-wide Python files after tranche 10: `81`
- Repository-wide Python files after tranche 11: `71`
- Repository-wide Python files after tranche 12: `68`
- Repository-wide Python files after tranche 13: `67`
- Net repository-wide Python reduction: `41`
- `bigclaw-go` Go files before: `267`
- `bigclaw-go` Go files after: `267`
- Net Go file reduction: `0`
- Root `pyproject.toml`: absent before and after
- Root `setup.py`: absent before and after

### Validation Record

- `rg -n "repo_board|repo_commits|repo_gateway|repo_governance|repo_registry|repo_triage" src tests || true`
  - Result: no matches
- `PYTHONPATH=src python3 -m pytest tests/test_observability.py tests/test_repo_links.py tests/test_repo_rollout.py tests/test_repo_collaboration.py`
  - Result: `11 passed in 0.17s`
- `rg -n "github_sync|parallel_refill|workspace_bootstrap_cli" src tests || true`
  - Result: one expected match in `src/bigclaw/legacy_shim.py` for the retained
    shim-builder function name `build_github_sync_args`; no deleted module
    imports remain
- `PYTHONPATH=src python3 -m pytest tests/test_workspace_bootstrap.py tests/test_observability.py tests/test_repo_links.py tests/test_repo_rollout.py tests/test_repo_collaboration.py`
  - Result: `20 passed in 3.14s`
- `PYTHONPATH=src python3 - <<'PY'` with `import bigclaw`
  - Result: `import ok`
- `PYTHONPATH=src python3 -m pytest tests/test_workspace_bootstrap.py tests/test_observability.py tests/test_repo_links.py tests/test_repo_rollout.py tests/test_repo_collaboration.py`
  - Result: `20 passed in 3.16s`
- `rg -n "roadmap|validation_policy" src tests || true`
  - Result: no matches
- `PYTHONPATH=src python3 - <<'PY'` with `import bigclaw`
  - Result: `import ok`
- `PYTHONPATH=src python3 -m pytest tests/test_workspace_bootstrap.py tests/test_observability.py tests/test_repo_links.py tests/test_repo_rollout.py tests/test_repo_collaboration.py`
  - Result: `20 passed in 3.20s`
- `rg -n "dashboard_run_contract|DashboardRunContract|DashboardRunContractAudit|DashboardRunContractLibrary|SchemaField|SurfaceSchema" src tests || true`
  - Result: no matches
- `PYTHONPATH=src python3 - <<'PY'` with `import bigclaw`
  - Result: `import ok`
- `PYTHONPATH=src python3 -m pytest tests/test_workspace_bootstrap.py tests/test_observability.py tests/test_repo_links.py tests/test_repo_rollout.py tests/test_repo_collaboration.py`
  - Result: `20 passed in 3.17s`
- `rg -n "memory.py|TaskMemoryStore|MemoryPattern|from \\.memory|from bigclaw\\.memory|import bigclaw\\.memory" src tests || true`
  - Result: no matches
- `PYTHONPATH=src python3 - <<'PY'` with `import bigclaw`
  - Result: `import ok`
- `PYTHONPATH=src python3 -m pytest tests/test_workspace_bootstrap.py tests/test_observability.py tests/test_repo_links.py tests/test_repo_rollout.py tests/test_repo_collaboration.py`
  - Result: `20 passed in 3.67s`
- `rg -n "from \\.connectors|from bigclaw\\.connectors|from \\.mapping|from bigclaw\\.mapping|map_source_issue_to_task|SourceIssue|GitHubConnector|LinearConnector|JiraConnector" src tests || true`
  - Result: no matches
- `PYTHONPATH=src python3 - <<'PY'` with `import bigclaw`
  - Result: `import ok`
- `PYTHONPATH=src python3 -m pytest tests/test_workspace_bootstrap.py tests/test_observability.py tests/test_repo_links.py tests/test_repo_rollout.py tests/test_repo_collaboration.py`
  - Result: `20 passed in 3.09s`
- `rg -n "pilot.py|PilotKPI|PilotImplementationResult|render_pilot_implementation_report|from \\.pilot|from bigclaw\\.pilot|import bigclaw\\.pilot" src tests || true`
  - Result: no matches
- `PYTHONPATH=src python3 - <<'PY'` with `import bigclaw`
  - Result: `import ok`
- `PYTHONPATH=src python3 -m pytest tests/test_workspace_bootstrap.py tests/test_observability.py tests/test_repo_links.py tests/test_repo_rollout.py tests/test_repo_collaboration.py`
  - Result: `20 passed in 3.02s`
- `rg -n "from \\.design_system|from \\.console_ia|from \\.saved_views|from \\.ui_review|from \\.dsl|from bigclaw\\.(design_system|console_ia|saved_views|ui_review|dsl)|WorkflowDefinition|SavedView|ConsoleIA|DesignSystem|UIReviewPack" src tests || true`
  - Result: no matches
- `PYTHONPATH=src python3 - <<'PY'` with `import bigclaw`
  - Result: `import ok`
- `PYTHONPATH=src python3 -m pytest tests/test_workspace_bootstrap.py tests/test_observability.py tests/test_repo_links.py tests/test_repo_rollout.py tests/test_repo_collaboration.py`
  - Result: `20 passed in 3.06s`
- `rg -n "workspace_bootstrap|workspace_bootstrap_validation|bootstrap_workspace|cleanup_workspace|build_validation_report" src tests || true`
  - Result: one expected match in `src/bigclaw/legacy_shim.py` for the retained
    shim-builder function name `build_workspace_bootstrap_args`; no deleted
    module imports remain
- `PYTHONPATH=src python3 - <<'PY'` with `import bigclaw`
  - Result: `import ok`
- `PYTHONPATH=src python3 -m pytest tests/test_observability.py tests/test_repo_links.py tests/test_repo_rollout.py tests/test_repo_collaboration.py`
  - Result: `11 passed in 0.06s`
- `cd bigclaw-go && go test ./internal/legacyshim ./cmd/bigclawctl ./internal/regression -run 'Test(FrozenCompileCheckFilesUsesFrozenShimList|CompileCheckRunsPyCompileAgainstFrozenShimList|RunLegacyPythonCompileCheckJSONOutputDoesNotEscapeArrowTokens|LegacyMainlineCompatibilityManifestStaysAligned)'`
  - Result: `ok` for `./internal/legacyshim`, `./cmd/bigclawctl`, and `./internal/regression`
- `rg -n "src/bigclaw/__main__\\.py|python -m bigclaw|python -m bigclaw serve" bigclaw-go docs src tests scripts || true`
  - Result: remaining matches are historical issue-pack entries plus the
    retained `warn_legacy_service_surface` default string in `src/bigclaw/runtime.py`;
    no live Python entrypoint file remains

### Follow-on tranche 14

- `src/bigclaw/repo_links.py`
  - Deleted.
  - Reason: after inlining `RunCommitBinding`, role validation, and
    `bind_run_commits` into `src/bigclaw/observability.py`, the module had no
    remaining in-repo consumers.
- `src/bigclaw/repo_plane.py`
  - Deleted.
  - Reason: `RepoSpace` and `RepoAgent` had no surviving in-repo references,
    and `RunCommitLink` now lives in `src/bigclaw/observability.py`.
- `src/bigclaw/observability.py`
  - Updated.
  - Reason: it now owns the run-commit link model and binding helpers used by
    `TaskRun.record_closeout`.
- `tests/test_observability.py`
  - Updated.
  - Reason: import `RunCommitLink` from `bigclaw.observability`.
- `tests/test_repo_links.py`
  - Updated.
  - Reason: import `RunCommitLink` and `bind_run_commits` from
    `bigclaw.observability`.

### Follow-on tranche 14 inventory

- `src/bigclaw` Python files after tranche 14: `17`
- Repository-wide Python files after tranche 14: `65`
- Net repository-wide Python reduction: `43`
- `bigclaw-go` Go files after tranche 14: `267`
- Root `pyproject.toml`: absent
- Root `setup.py`: absent

### Follow-on tranche 14 validation

- `python3 -m py_compile src/bigclaw/observability.py tests/test_repo_links.py tests/test_observability.py`
  - Result: success
- `PYTHONPATH=src python3 -m pytest tests/test_observability.py tests/test_repo_links.py tests/test_repo_rollout.py tests/test_repo_collaboration.py`
  - Result: `11 passed in 0.06s`
- `PYTHONPATH=src python3 - <<'PY'` with `import bigclaw`
  - Result: `import ok`
- `find src/bigclaw -type f -name '*.py' | wc -l`
  - Result: `17`
- `find . -type f -name '*.py' | wc -l`
  - Result: `65`
- `find bigclaw-go -type f -name '*.go' | wc -l`
  - Result: `267`
- `test -f pyproject.toml && echo present || echo absent`
  - Result: `absent`
- `test -f setup.py && echo present || echo absent`
  - Result: `absent`
- `git diff --check`
  - Result: success

### Follow-on tranche 15

- `src/bigclaw/deprecation.py`
  - Deleted.
  - Reason: its warning helpers were only used by `runtime.py` and
    `scripts/dev_smoke.py`, so the migration-only surface could be absorbed
    into `src/bigclaw/runtime.py` without changing behavior.
- `src/bigclaw/runtime.py`
  - Updated.
  - Reason: now owns `LEGACY_RUNTIME_GUIDANCE`,
    `legacy_runtime_message`, and `warn_legacy_runtime_surface` directly.
- `scripts/dev_smoke.py`
  - Updated.
  - Reason: imports `warn_legacy_runtime_surface` from `bigclaw.runtime`
    instead of the deleted module.

### Follow-on tranche 15 inventory

- `src/bigclaw` Python files after tranche 15: `16`
- Repository-wide Python files after tranche 15: `64`
- Net repository-wide Python reduction: `44`
- `bigclaw-go` Go files after tranche 15: `267`
- Root `pyproject.toml`: absent
- Root `setup.py`: absent

### Follow-on tranche 15 validation

- `rg -n "bigclaw\\.deprecation|from \\.deprecation" src tests scripts`
  - Result: no matches
- `python3 -m py_compile src/bigclaw/runtime.py scripts/dev_smoke.py tests/test_runtime_matrix.py tests/test_reports.py tests/test_evaluation.py`
  - Result: success
- `PYTHONPATH=src python3 -m pytest tests/test_runtime_matrix.py tests/test_reports.py tests/test_evaluation.py`
  - Result: `44 passed in 0.18s`
- `PYTHONPATH=src python3 scripts/dev_smoke.py --help`
  - Result: emitted the expected deprecation warning and `bigclawctl dev-smoke`
    usage text; exited successfully
- `PYTHONPATH=src python3 - <<'PY'` with `import bigclaw` and `import bigclaw.runtime`
  - Result: `import ok`
- `find src/bigclaw -type f -name '*.py' | wc -l`
  - Result: `16`
- `find . -type f -name '*.py' | wc -l`
  - Result: `64`
- `find bigclaw-go -type f -name '*.go' | wc -l`
  - Result: `267`
- `test -f pyproject.toml && echo present || echo absent`
  - Result: `absent`
- `test -f setup.py && echo present || echo absent`
  - Result: `absent`
- `git diff --check`
  - Result: success

### Follow-on tranche 16

- `src/bigclaw/risk.py`
  - Deleted.
  - Reason: its scorer surface only fed `runtime.py`, a dedicated test slice,
    and package exports, so the behavior could move into `runtime.py` without
    widening the active module graph.
- `src/bigclaw/runtime.py`
  - Updated.
  - Reason: now owns `RiskFactor`, `RiskScore`, and `RiskScorer` directly.
- `src/bigclaw/__init__.py`
  - Updated.
  - Reason: exports the moved risk symbols from `runtime.py` and installs a
    `bigclaw.risk` compatibility module backed by the runtime surface.

### Follow-on tranche 16 inventory

- `src/bigclaw` Python files after tranche 16: `15`
- Repository-wide Python files after tranche 16: `63`
- Net repository-wide Python reduction: `45`
- `bigclaw-go` Go files after tranche 16: `267`
- Root `pyproject.toml`: absent
- Root `setup.py`: absent

### Follow-on tranche 16 validation

- `python3 -m py_compile src/bigclaw/runtime.py src/bigclaw/__init__.py tests/test_risk.py tests/test_runtime_matrix.py`
  - Result: success
- `PYTHONPATH=src python3 -m pytest tests/test_risk.py tests/test_runtime_matrix.py`
  - Result: `6 passed in 0.05s`
- `PYTHONPATH=src python3 - <<'PY'` with `import bigclaw`, `from bigclaw.risk import RiskScorer`, and `from bigclaw.scheduler import Scheduler`
  - Result: `import ok RiskScorer Scheduler`
- `find src/bigclaw -type f -name '*.py' | wc -l`
  - Result: `15`
- `find . -type f -name '*.py' | wc -l`
  - Result: `63`
- `find bigclaw-go -type f -name '*.go' | wc -l`
  - Result: `267`
- `test -f pyproject.toml && echo present || echo absent`
  - Result: `absent`
- `test -f setup.py && echo present || echo absent`
  - Result: `absent`
- `git diff --check`
  - Result: success

### Follow-on tranche 17

- `src/bigclaw/event_bus.py`
  - Deleted.
  - Reason: its event routing surface only depended on observability state,
    dedicated tests, and package exports, so it could be absorbed into
    `src/bigclaw/observability.py`.
- `src/bigclaw/observability.py`
  - Updated.
  - Reason: now owns `BusEvent`, `EventBus`, and the event type constants.
- `src/bigclaw/__init__.py`
  - Updated.
  - Reason: installs a `bigclaw.event_bus` compatibility module backed by the
    observability surface and re-exports the moved symbols from there.

### Follow-on tranche 17 inventory

- `src/bigclaw` Python files after tranche 17: `14`
- Repository-wide Python files after tranche 17: `62`
- Net repository-wide Python reduction: `46`
- `bigclaw-go` Go files after tranche 17: `267`
- Root `pyproject.toml`: absent
- Root `setup.py`: absent

### Follow-on tranche 17 validation

- `python3 -m py_compile src/bigclaw/observability.py src/bigclaw/__init__.py tests/test_event_bus.py tests/test_observability.py`
  - Result: success
- `PYTHONPATH=src python3 -m pytest tests/test_event_bus.py tests/test_observability.py`
  - Result: `10 passed in 0.05s`
- `PYTHONPATH=src python3 - <<'PY'` with `from bigclaw.event_bus import EventBus, BusEvent, PULL_REQUEST_COMMENT_EVENT` and `from bigclaw.observability import EventBus as ObsEventBus`
  - Result: `EventBus BusEvent pull_request.comment EventBus`
- `find src/bigclaw -type f -name '*.py' | wc -l`
  - Result: `14`
- `find . -type f -name '*.py' | wc -l`
  - Result: `62`
- `find bigclaw-go -type f -name '*.go' | wc -l`
  - Result: `267`
- `test -f pyproject.toml && echo present || echo absent`
  - Result: `absent`
- `test -f setup.py && echo present || echo absent`
  - Result: `absent`
- `git diff --check`
  - Result: success

### Follow-on tranche 18

- `src/bigclaw/governance.py`
  - Deleted.
  - Reason: its scope-freeze surface only fed `planning.py`, dedicated tests,
    and package exports, so it could be absorbed into `src/bigclaw/planning.py`.
- `src/bigclaw/planning.py`
  - Updated.
  - Reason: now owns `FreezeException`, `GovernanceBacklogItem`,
    `ScopeFreezeBoard`, `ScopeFreezeAudit`, `ScopeFreezeGovernance`, and
    `render_scope_freeze_report`.
- `src/bigclaw/__init__.py`
  - Updated.
  - Reason: installs a `bigclaw.governance` compatibility module backed by the
    planning surface and re-exports the moved governance symbols from there.

### Follow-on tranche 18 inventory

- `src/bigclaw` Python files after tranche 18: `13`
- Repository-wide Python files after tranche 18: `61`
- Net repository-wide Python reduction: `47`
- `bigclaw-go` Go files after tranche 18: `267`
- Root `pyproject.toml`: absent
- Root `setup.py`: absent

### Follow-on tranche 18 validation

- `python3 -m py_compile src/bigclaw/planning.py src/bigclaw/__init__.py tests/test_governance.py tests/test_planning.py tests/test_repo_rollout.py`
  - Result: success
- `PYTHONPATH=src python3 -m pytest tests/test_governance.py tests/test_planning.py tests/test_repo_rollout.py`
  - Result: `20 passed in 0.07s`
- `PYTHONPATH=src python3 - <<'PY'` with `from bigclaw.governance import ScopeFreezeAudit, ScopeFreezeGovernance` and `from bigclaw.planning import ScopeFreezeAudit as PlanningScopeFreezeAudit`
  - Result: `ScopeFreezeAudit ScopeFreezeGovernance ScopeFreezeAudit`
- `find src/bigclaw -type f -name '*.py' | wc -l`
  - Result: `13`
- `find . -type f -name '*.py' | wc -l`
  - Result: `61`
- `find bigclaw-go -type f -name '*.go' | wc -l`
  - Result: `267`
- `test -f pyproject.toml && echo present || echo absent`
  - Result: `absent`
- `test -f setup.py && echo present || echo absent`
  - Result: `absent`
- `git diff --check`
  - Result: success

### Follow-on tranche 19

- `src/bigclaw/execution_contract.py`
  - Deleted.
  - Reason: the execution-contract surface had no remaining in-repo runtime
    consumers beyond tests and package exports, and its concrete builder is an
    operations contract now owned in `src/bigclaw/operations.py`.
- `src/bigclaw/operations.py`
  - Updated.
  - Reason: now owns the execution-contract dataclasses, audit library,
    permission matrix, report renderer, and `build_operations_api_contract`.
- `src/bigclaw/__init__.py`
  - Updated.
  - Reason: installs a `bigclaw.execution_contract` compatibility module backed
    by the operations surface and re-exports the moved contract symbols from
    there.

### Follow-on tranche 19 inventory

- `src/bigclaw` Python files after tranche 19: `12`
- Repository-wide Python files after tranche 19: `60`
- Net repository-wide Python reduction: `48`
- `bigclaw-go` Go files after tranche 19: `267`
- Root `pyproject.toml`: absent
- Root `setup.py`: absent

### Follow-on tranche 19 validation

- `python3 -m py_compile src/bigclaw/operations.py src/bigclaw/__init__.py tests/test_execution_contract.py tests/test_operations.py`
  - Result: success
- `PYTHONPATH=src python3 -m pytest tests/test_execution_contract.py tests/test_operations.py`
  - Result: `27 passed in 0.06s`
- `PYTHONPATH=src python3 - <<'PY'` with `from bigclaw.execution_contract import build_operations_api_contract, ExecutionContractLibrary`
  - Result: last API entry is `GET /operations/billing/entitlements` and `audit.release_ready` is `True`
- `find src/bigclaw -type f -name '*.py' | wc -l`
  - Result: `12`
- `find . -type f -name '*.py' | wc -l`
  - Result: `60`
- `find bigclaw-go -type f -name '*.go' | wc -l`
  - Result: `267`
- `test -f pyproject.toml && echo present || echo absent`
  - Result: `absent`
- `test -f setup.py && echo present || echo absent`
  - Result: `absent`
- `git diff --check`
  - Result: success
