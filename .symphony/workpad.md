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
- Inline the small `src/bigclaw/run_detail.py` rendering helpers into
  `src/bigclaw/reports.py` if the import graph stays acyclic, then preserve
  `bigclaw.run_detail` as a compatibility module.
- Inline the smaller `src/bigclaw/audit_events.py` event-spec surface into
  `src/bigclaw/observability.py` first, since its active consumers already sit
  on the observability/runtime path.
- Active tranche: fold `src/bigclaw/run_detail.py` into `src/bigclaw/reports.py`,
  repoint `evaluation.py`, preserve `bigclaw.run_detail` via package shim, and
  delete the residual leaf module.

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

### Follow-on tranche 20

- `src/bigclaw/audit_events.py`
  - Deleted.
  - Reason: the audit-event spec surface only fed observability/runtime plus
    dedicated tests and package exports, so it could be absorbed directly into
    `src/bigclaw/observability.py`.
- `src/bigclaw/observability.py`
  - Updated.
  - Reason: now owns the audit event constants, `AuditEventSpec`,
    `P0_AUDIT_EVENT_SPECS`, `get_audit_event_spec`, and
    `missing_required_fields`.
- `src/bigclaw/runtime.py`
  - Updated.
  - Reason: imports audit event constants from observability instead of the
    deleted module.
- `src/bigclaw/reports.py`
  - Updated.
  - Reason: imports `FLOW_HANDOFF_EVENT` and `MANUAL_TAKEOVER_EVENT` directly
    from observability instead of relying on the deleted module path.
- `src/bigclaw/__init__.py`
  - Updated.
  - Reason: installs a `bigclaw.audit_events` compatibility module backed by
    observability and re-exports the moved audit-event symbols from there.

### Follow-on tranche 20 inventory

- `src/bigclaw` Python files after tranche 20: `11`
- Repository-wide Python files after tranche 20: `59`
- Net repository-wide Python reduction: `49`
- `bigclaw-go` Go files after tranche 20: `267`
- Root `pyproject.toml`: absent
- Root `setup.py`: absent

### Follow-on tranche 20 validation

- `rg -n "from \\.audit_events|from bigclaw\\.audit_events|import bigclaw\\.audit_events" src tests scripts`
  - Result: only the expected compatibility import remains in `tests/test_audit_events.py`
- `python3 -m py_compile src/bigclaw/observability.py src/bigclaw/runtime.py src/bigclaw/reports.py src/bigclaw/__init__.py tests/test_audit_events.py tests/test_observability.py tests/test_runtime_matrix.py`
  - Result: success
- `PYTHONPATH=src python3 -m pytest tests/test_audit_events.py tests/test_observability.py tests/test_runtime_matrix.py`
  - Result: `15 passed in 0.07s`
- `PYTHONPATH=src python3 - <<'PY'` with `from bigclaw.audit_events import APPROVAL_RECORDED_EVENT, missing_required_fields` and `from bigclaw.observability import AuditEventSpec`
  - Result: `execution.approval_recorded True AuditEventSpec`
- `find src/bigclaw -type f -name '*.py' | wc -l`
  - Result: `11`
- `find . -type f -name '*.py' | wc -l`
  - Result: `59`
- `find bigclaw-go -type f -name '*.go' | wc -l`
  - Result: `267`
- `test -f pyproject.toml && echo present || echo absent`
  - Result: `absent`
- `test -f setup.py && echo present || echo absent`
  - Result: `absent`
- `git diff --check`
  - Result: success

### Follow-on tranche 21

- `src/bigclaw/run_detail.py`
  - Deleted.
  - Reason: the run-detail rendering dataclasses and HTML helpers were only
    consumed by `reports.py`, `evaluation.py`, and package compatibility
    exports, so they could be folded into the active reports surface.
- `src/bigclaw/reports.py`
  - Updated.
  - Reason: now owns `RunDetailStat`, `RunDetailResource`,
    `RunDetailEvent`, `RunDetailTab`, `render_run_detail_console`,
    `render_resource_grid`, and `render_timeline_panel`.
- `src/bigclaw/evaluation.py`
  - Updated.
  - Reason: imports the run-detail surface from `reports.py` instead of the
    deleted leaf module.
- `src/bigclaw/__init__.py`
  - Updated.
  - Reason: installs a `bigclaw.run_detail` compatibility module backed by the
    reports surface and re-exports the moved run-detail symbols there.

### Follow-on tranche 21 inventory

- `src/bigclaw` Python files after tranche 21: `10`
- Repository-wide Python files after tranche 21: `58`
- Net repository-wide Python reduction: `50`
- `bigclaw-go` Go files after tranche 21: `267`
- Root `pyproject.toml`: absent
- Root `setup.py`: absent

### Follow-on tranche 21 validation

- `python3 -m py_compile src/bigclaw/reports.py src/bigclaw/evaluation.py src/bigclaw/__init__.py tests/test_reports.py tests/test_evaluation.py tests/test_observability.py`
  - Result: success
- `PYTHONPATH=src python3 -m pytest tests/test_reports.py tests/test_evaluation.py tests/test_observability.py`
  - Result: initial run failed during collection because `src/bigclaw/__init__.py`
    imported `reports.py` before the existing legacy `bigclaw.orchestration`
    shim was installed; after moving the `run_detail` shim installation to the
    normal post-import phase, the rerun passed with `48 passed in 0.08s`
- `PYTHONPATH=src python3 - <<'PY'` with `from bigclaw.run_detail import RunDetailStat, render_run_detail_console` and `from bigclaw.reports import RunDetailEvent, RunDetailTab`
  - Result: `status RunDetailEvent RunDetailTab True`
- `find src/bigclaw -type f -name '*.py' | wc -l`
  - Result: `10`
- `find . -type f -name '*.py' | wc -l`
  - Result: `58`
- `find bigclaw-go -type f -name '*.go' | wc -l`
  - Result: `267`
- `test -f pyproject.toml && echo present || echo absent`
  - Result: `absent`
- `test -f setup.py && echo present || echo absent`
  - Result: `absent`
- `git diff --check`
  - Result: success

### Follow-on tranche 22 queued

- Target `src/bigclaw/legacy_shim.py` next.
- Reason: it is an isolated migration helper only consumed by five legacy
  `scripts/ops/*` wrappers, so its small argument-translation and subprocess
  helpers can be inlined into those wrappers without touching the runtime /
  reporting graph.
- Planned edits:
  - Inline `append_missing_flag` and repo-root resolution into
    `scripts/ops/bigclaw_workspace_bootstrap.py`.
  - Inline workspace-validate argument translation into
    `scripts/ops/symphony_workspace_validate.py`.
  - Inline direct `bigclawctl` subprocess wrappers into
    `scripts/ops/symphony_workspace_bootstrap.py`,
    `scripts/ops/bigclaw_github_sync.py`, and
    `scripts/ops/bigclaw_refill_queue.py`.
  - Delete `src/bigclaw/legacy_shim.py`.

### Follow-on tranche 22

- `src/bigclaw/legacy_shim.py`
  - Deleted.
  - Reason: the module was only a migration helper for five legacy
    `scripts/ops/*` wrappers and had no remaining runtime or test ownership in
    `src/bigclaw/**`.
- `scripts/ops/bigclaw_workspace_bootstrap.py`
  - Updated.
  - Reason: now owns its minimal `append_missing_flag` helper and repo-root
    resolution locally, so it no longer imports `bigclaw.legacy_shim`.
- `scripts/ops/symphony_workspace_validate.py`
  - Updated.
  - Reason: now owns the small workspace-validate argument translation locally.
- `scripts/ops/symphony_workspace_bootstrap.py`
  - Updated.
  - Reason: now shells directly to `scripts/ops/bigclawctl workspace`.
- `scripts/ops/bigclaw_github_sync.py`
  - Updated.
  - Reason: now shells directly to `scripts/ops/bigclawctl github-sync`.
- `scripts/ops/bigclaw_refill_queue.py`
  - Updated.
  - Reason: now shells directly to `scripts/ops/bigclawctl refill`.

### Follow-on tranche 22 inventory

- `src/bigclaw` Python files after tranche 22: `9`
- Repository-wide Python files after tranche 22: `57`
- Net repository-wide Python reduction: `51`
- `bigclaw-go` Go files after tranche 22: `267`
- Root `pyproject.toml`: absent
- Root `setup.py`: absent

### Follow-on tranche 22 validation

- `python3 -m py_compile scripts/ops/bigclaw_workspace_bootstrap.py scripts/ops/symphony_workspace_validate.py scripts/ops/symphony_workspace_bootstrap.py scripts/ops/bigclaw_github_sync.py scripts/ops/bigclaw_refill_queue.py`
  - Result: success
- `rg -n "legacy_shim|workspace_validate_args|translate_workspace_validate_args|run_bigclawctl_shim|append_missing_flag|repo_root_from_script" src tests scripts`
  - Result: only the newly inlined local helper definitions/usages remain in
    the updated scripts; no `src/bigclaw/legacy_shim.py` imports remain
- `python3 scripts/ops/bigclaw_github_sync.py --help`
  - Result: `usage: bigclawctl github-sync <install|status|sync> [flags]`
- `python3 scripts/ops/bigclaw_refill_queue.py --help`
  - Result: refill usage text printed successfully, including `seed`
    subcommand and flags
- `python3 scripts/ops/symphony_workspace_bootstrap.py --help`
  - Result: `usage: bigclawctl workspace <bootstrap|cleanup|validate> [flags]`
- `python3 scripts/ops/symphony_workspace_validate.py --help`
  - Result: workspace-validate usage text printed successfully, including
    translated `--report` / `--cleanup` flag forms
- `python3 scripts/ops/bigclaw_workspace_bootstrap.py --help`
  - Result: `usage: bigclawctl workspace <bootstrap|cleanup|validate> [flags]`
- `find src/bigclaw -type f -name '*.py' | wc -l`
  - Result: `9`
- `find . -type f -name '*.py' | wc -l`
  - Result: `57`
- `find bigclaw-go -type f -name '*.go' | wc -l`
  - Result: `267`
- `test -f pyproject.toml && echo present || echo absent`
  - Result: `absent`
- `test -f setup.py && echo present || echo absent`
  - Result: `absent`
- `git diff --check`
  - Result: success

### Follow-on tranche 23 queued

- Target `src/bigclaw/collaboration.py` next.
- Reason: the collaboration surface is only consumed by
  `observability.py`, `reports.py`, package exports, and focused tests; moving
  it into `observability.py` keeps the import graph acyclic because
  `reports.py` already imports observability while observability does not
  import reports.
- Planned edits:
  - Move the collaboration dataclasses and helper/render functions into
    `src/bigclaw/observability.py`.
  - Repoint `src/bigclaw/reports.py` to import collaboration symbols from
    observability.
  - Install a `bigclaw.collaboration` compatibility shim in
    `src/bigclaw/__init__.py`.
  - Delete `src/bigclaw/collaboration.py`.

### Follow-on tranche 23

- `src/bigclaw/collaboration.py`
  - Deleted.
  - Reason: the collaboration surface only fed observability, reports, package
    exports, and focused tests, so it could be absorbed into the active
    observability module without creating an import cycle.
- `src/bigclaw/observability.py`
  - Updated.
  - Reason: now owns `CollaborationComment`, `DecisionNote`,
    `CollaborationThread`, `build_collaboration_thread`,
    `merge_collaboration_threads`, `build_collaboration_thread_from_audits`,
    `render_collaboration_lines`, and `render_collaboration_panel_html`.
- `src/bigclaw/reports.py`
  - Updated.
  - Reason: imports the collaboration surface from observability instead of the
    deleted module.
- `src/bigclaw/__init__.py`
  - Updated.
  - Reason: installs a `bigclaw.collaboration` compatibility module backed by
    observability and re-exports the moved collaboration symbols there.

### Follow-on tranche 23 inventory

- `src/bigclaw` Python files after tranche 23: `8`
- Repository-wide Python files after tranche 23: `56`
- Net repository-wide Python reduction: `52`
- `bigclaw-go` Go files after tranche 23: `267`
- Root `pyproject.toml`: absent
- Root `setup.py`: absent

### Follow-on tranche 23 validation

- `python3 -m py_compile src/bigclaw/observability.py src/bigclaw/reports.py src/bigclaw/__init__.py tests/test_observability.py tests/test_reports.py tests/test_repo_collaboration.py`
  - Result: success
- `PYTHONPATH=src python3 -m pytest tests/test_observability.py tests/test_reports.py tests/test_repo_collaboration.py`
  - Result: `42 passed in 0.07s`
- `PYTHONPATH=src python3 - <<'PY'` with `from bigclaw.collaboration import CollaborationComment, merge_collaboration_threads` and `from bigclaw.observability import DecisionNote, build_collaboration_thread`
  - Result: `CollaborationComment merged`
- `find src/bigclaw -type f -name '*.py' | wc -l`
  - Result: `8`
- `find . -type f -name '*.py' | wc -l`
  - Result: `56`
- `find bigclaw-go -type f -name '*.go' | wc -l`
  - Result: `267`
- `test -f pyproject.toml && echo present || echo absent`
  - Result: `absent`
- `test -f setup.py && echo present || echo absent`
  - Result: `absent`
- `git diff --check`
  - Result: success

### Follow-on tranche 24 queued

- Target `src/bigclaw/evaluation.py` next.
- Reason: the evaluation surface is now a leaf on top of reports,
  observability, scheduler, and models; `reports.py` does not depend on
  evaluation, and `operations.py` only imports `BenchmarkSuiteResult`, so the
  move can stay acyclic by repointing those imports into `reports.py`.
- Planned edits:
  - Move the benchmark/evaluation dataclasses, runner, and report renderers
    into `src/bigclaw/reports.py`.
  - Repoint `src/bigclaw/operations.py` and package exports to the reports
    surface.
  - Install a `bigclaw.evaluation` compatibility shim in
    `src/bigclaw/__init__.py`.
  - Delete `src/bigclaw/evaluation.py`.

### Follow-on tranche 24

- `src/bigclaw/evaluation.py`
  - Deleted.
  - Reason: the benchmark/evaluation surface was a leaf on top of reports,
    observability, scheduler, and models, so it could be absorbed into the
    active reports surface without creating a cycle.
- `src/bigclaw/reports.py`
  - Updated.
  - Reason: now owns `EvaluationCriterion`, `BenchmarkCase`,
    `ReplayRecord`, `ReplayOutcome`, `BenchmarkResult`,
    `BenchmarkComparison`, `BenchmarkSuiteResult`, `BenchmarkRunner`,
    `render_benchmark_suite_report`, `render_replay_detail_page`, and
    `render_run_replay_index_page`.
- `src/bigclaw/operations.py`
  - Updated.
  - Reason: imports `BenchmarkSuiteResult` from reports instead of the deleted
    evaluation module.
- `src/bigclaw/__init__.py`
  - Updated.
  - Reason: installs a `bigclaw.evaluation` compatibility module backed by the
    reports surface and re-exports the moved evaluation symbols there.
- `src/bigclaw/planning.py`
  - Updated.
  - Reason: repointed rollback-simulation evidence metadata from the deleted
    `evaluation.py` module to the folded `reports.py` surface.
- `tests/test_planning.py`
  - Updated.
  - Reason: now asserts the retained `src/bigclaw/reports.py` evidence target
    instead of the deleted `src/bigclaw/evaluation.py` path.

### Follow-on tranche 24 inventory

- `src/bigclaw` Python files after tranche 24: `7`
- Repository-wide Python files after tranche 24: `55`
- Net repository-wide Python reduction: `53`
- `bigclaw-go` Go files after tranche 24: `267`
- Root `pyproject.toml`: absent
- Root `setup.py`: absent

### Follow-on tranche 24 validation

- `python3 -m py_compile src/bigclaw/reports.py src/bigclaw/operations.py src/bigclaw/__init__.py tests/test_evaluation.py tests/test_reports.py tests/test_operations.py`
  - Result: success
- `PYTHONPATH=src python3 -m pytest tests/test_evaluation.py tests/test_reports.py tests/test_operations.py`
  - Result: `61 passed in 0.09s`
- `PYTHONPATH=src python3 - <<'PY'` with `from bigclaw.evaluation import BenchmarkRunner, render_benchmark_suite_report` and `from bigclaw.reports import BenchmarkSuiteResult`
  - Result: `BenchmarkRunner True 0`
- `PYTHONPATH=src python3 -m pytest tests/test_evaluation.py tests/test_operations.py tests/test_observability.py tests/test_repo_links.py tests/test_repo_rollout.py tests/test_repo_collaboration.py`
  - Result: `38 passed in 0.08s`
- `PYTHONPATH=src python3 - <<'PY'` with `import bigclaw`, `import bigclaw.evaluation`, and inspection of `bigclaw.evaluation.BenchmarkRunner.__module__`
  - Result: `import ok`, module=`bigclaw.reports`, shim attr present=`True`
- `PYTHONPATH=src python3 -m pytest tests/test_evaluation.py tests/test_operations.py tests/test_observability.py tests/test_repo_links.py tests/test_repo_rollout.py tests/test_repo_collaboration.py tests/test_planning.py`
  - Result: `52 passed in 0.10s`
- `find src/bigclaw -type f -name '*.py' | wc -l`
  - Result: `7`
- `find . -type f -name '*.py' | wc -l`
  - Result: `55`
- `find bigclaw-go -type f -name '*.go' | wc -l`
  - Result: `267`
- `test -f pyproject.toml && echo present || echo absent`
  - Result: `absent`
- `test -f setup.py && echo present || echo absent`
  - Result: `absent`
- `git diff --check`
  - Result: success

### Follow-on tranche 25 queued

- Target `src/bigclaw/models.py` next.
- Reason: the remaining model surface is a leaf of enums/dataclasses consumed by
  observability, runtime, reports, operations, package exports, and focused
  tests; `observability.py` can absorb those types without creating an import
  cycle because it only depended on `Task` from the deleted module.
- Planned edits:
  - Move the model enums/dataclasses from `src/bigclaw/models.py` into
    `src/bigclaw/observability.py`.
  - Repoint `runtime.py`, `reports.py`, `operations.py`, and package exports to
    the observability-owned model surface.
  - Install a `bigclaw.models` compatibility shim in `src/bigclaw/__init__.py`.
  - Delete `src/bigclaw/models.py`.

### Follow-on tranche 25

- `src/bigclaw/models.py`
  - Deleted.
  - Reason: the remaining model enums/dataclasses were pure shared types with
    no behavior beyond serialization helpers, so they could be absorbed into
    the active observability surface without introducing an import cycle.
- `src/bigclaw/observability.py`
  - Updated.
  - Reason: now owns the former `models.py` enums and dataclasses:
    `TaskState`, `RiskLevel`, `Priority`, `TriageStatus`, `FlowTrigger`,
    `FlowRunStatus`, `FlowStepStatus`, `BillingInterval`, `Task`,
    `RiskSignal`, `RiskAssessment`, `TriageLabel`, `TriageRecord`,
    `FlowTemplateStep`, `FlowTemplate`, `FlowStepRun`, `FlowRun`,
    `BillingRate`, `UsageRecord`, and `BillingSummary`.
- `src/bigclaw/runtime.py`
  - Updated.
  - Reason: now imports `Priority`, `RiskLevel`, and `Task` from
    observability instead of the deleted models module.
- `src/bigclaw/reports.py`
  - Updated.
  - Reason: now imports `Task` and other shared model types from
    observability instead of the deleted models module.
- `src/bigclaw/operations.py`
  - Updated.
  - Reason: now imports `Task` from observability instead of the deleted
    models module.
- `src/bigclaw/__init__.py`
  - Updated.
  - Reason: package exports now source model symbols from observability and
    install a `bigclaw.models` compatibility module backed by that surface.

### Follow-on tranche 25 inventory

- `src/bigclaw` Python files after tranche 25: `6`
- Repository-wide Python files after tranche 25: `54`
- Net repository-wide Python reduction: `54`
- `bigclaw-go` Go files after tranche 25: `267`
- Root `pyproject.toml`: absent
- Root `setup.py`: absent

### Follow-on tranche 25 validation

- `python3 -m py_compile src/bigclaw/observability.py src/bigclaw/runtime.py src/bigclaw/reports.py src/bigclaw/operations.py src/bigclaw/__init__.py tests/test_models.py tests/test_observability.py tests/test_runtime_matrix.py tests/test_reports.py tests/test_operations.py`
  - Result: success
- `PYTHONPATH=src python3 - <<'PY'` with `import bigclaw`, `import bigclaw.models`, and inspection of `Task.__module__` / `Priority.P0.value`
  - Result: `import ok`, module=`bigclaw.observability`, `0`
- `PYTHONPATH=src python3 -m pytest tests/test_models.py tests/test_observability.py tests/test_runtime_matrix.py tests/test_reports.py tests/test_operations.py`
  - Result: `68 passed in 0.09s`
- `PYTHONPATH=src python3 -m pytest tests/test_scheduler.py tests/test_orchestration.py tests/test_queue.py tests/test_risk.py tests/test_event_bus.py tests/test_control_center.py tests/test_audit_events.py tests/test_repo_links.py tests/test_evaluation.py`
  - Result: `35 passed in 0.11s`
- `find src/bigclaw -type f -name '*.py' | wc -l`
  - Result: `6`
- `find . -type f -name '*.py' | wc -l`
  - Result: `54`

### Follow-on tranche 26 queued

- Target `src/bigclaw/planning.py` next.
- Reason: the remaining planning surface is isolated from the runtime/reporting
  core, only feeds dedicated tests, package exports, and the existing
  `bigclaw.governance`/`bigclaw.planning` compatibility surfaces, and can be
  merged into `src/bigclaw/operations.py` without creating an import cycle.
- Planned edits:
  - Move the planning/governance dataclasses, builders, and render helpers from
    `src/bigclaw/planning.py` into `src/bigclaw/operations.py`.
  - Repoint package exports plus `bigclaw.governance` and `bigclaw.planning`
    compatibility modules to the operations-owned surface.
  - Delete `src/bigclaw/planning.py`.

### Follow-on tranche 26

- `src/bigclaw/planning.py`
  - Deleted.
  - Reason: the remaining planning/governance surface was isolated from the
    runtime/reporting core and only served dedicated tests, package exports,
    and compatibility shims, so it could be absorbed into operations without
    creating an import cycle.
- `src/bigclaw/operations.py`
  - Updated.
  - Reason: now owns the former planning/governance dataclasses, builders, and
    render helpers, including `ScopeFreeze*`, candidate backlog/entry-gate
    types, rollout helpers, and the four-week execution plan surface.
- `src/bigclaw/__init__.py`
  - Updated.
  - Reason: package imports now source planning/governance symbols from
    operations and install `bigclaw.planning` plus `bigclaw.governance`
    compatibility modules backed by that surface.

### Follow-on tranche 26 inventory

- `src/bigclaw` Python files after tranche 26: `5`
- Repository-wide Python files after tranche 26: `53`
- Net repository-wide Python reduction: `55`
- `bigclaw-go` Go files after tranche 26: `267`
- Root `pyproject.toml`: absent
- Root `setup.py`: absent

### Follow-on tranche 26 validation

- `python3 -m py_compile src/bigclaw/operations.py src/bigclaw/__init__.py tests/test_planning.py tests/test_governance.py tests/test_repo_rollout.py tests/test_operations.py`
  - Result: success
- `PYTHONPATH=src python3 - <<'PY'` with `import bigclaw`, `import bigclaw.planning`, and `import bigclaw.governance`
  - Result: `import ok`; `bigclaw.planning.CandidatePlanner.__module__ == "bigclaw.operations"`; `bigclaw.governance.ScopeFreezeAudit.__module__ == "bigclaw.operations"`
- `PYTHONPATH=src python3 -m pytest tests/test_planning.py tests/test_governance.py tests/test_repo_rollout.py`
  - Result: `20 passed in 0.05s`
- `PYTHONPATH=src python3 -m pytest tests/test_operations.py tests/test_control_center.py tests/test_evaluation.py`
  - Result: `30 passed in 0.07s`
- `find src/bigclaw -type f -name '*.py' | wc -l`
  - Result: `5`
- `find . -type f -name '*.py' | wc -l`
  - Result: `53`

### Continuation turn 20 validation refresh

- `python3 -m py_compile src/bigclaw/operations.py src/bigclaw/__init__.py tests/test_planning.py tests/test_governance.py tests/test_repo_rollout.py tests/test_operations.py`
  - Result: success
- `PYTHONPATH=src python3 -m pytest tests/test_planning.py tests/test_governance.py tests/test_repo_rollout.py`
  - Result: `20 passed in 0.05s`
- `PYTHONPATH=src python3 -m pytest tests/test_operations.py tests/test_control_center.py tests/test_evaluation.py`
  - Result: `30 passed in 0.07s`
- `PYTHONPATH=src python3 - <<'PY'` with `import bigclaw`, `import bigclaw.planning`, and `import bigclaw.governance`
  - Result: `import ok`; `bigclaw.planning.CandidatePlanner.__module__ == "bigclaw.operations"`; `bigclaw.governance.ScopeFreezeAudit.__module__ == "bigclaw.operations"`
- `find src/bigclaw -type f -name '*.py' | wc -l`
  - Result: `5`
- `find . -type f -name '*.py' | wc -l`
  - Result: `53`
- `find bigclaw-go -type f -name '*.go' | wc -l`
  - Result: `267`
- `test -f pyproject.toml && echo present || echo absent`
  - Result: `absent`
- `test -f setup.py && echo present || echo absent`
  - Result: `absent`
- `git diff --check`
  - Result: success

### Follow-on tranche 27 queued

- Target `src/bigclaw/reports.py` next.
- Reason: the remaining report/rendering surface has no direct import cycle back
  into `operations.py` beyond the current one-way dependency, and most of its
  consumers already route through package exports or runtime deferred imports,
  so it can be merged into `src/bigclaw/operations.py` and preserved via a
  package-installed `bigclaw.reports` compatibility shim.
- Planned edits:
  - Move the report dataclasses, renderers, writers, replay helpers, and repo
    narrative exports from `src/bigclaw/reports.py` into
    `src/bigclaw/operations.py`.
  - Repoint package exports so `bigclaw.reports` resolves to the
    operations-owned surface.
  - Delete `src/bigclaw/reports.py`.

### Follow-on tranche 27

- `src/bigclaw/reports.py`
  - Deleted.
  - Reason: the remaining report/rendering surface was a leaf relative to the
    runtime/observability core and could be absorbed into operations while
    preserving `bigclaw.reports`, `bigclaw.evaluation`, and
    `bigclaw.run_detail` through package-installed compatibility shims.
- `src/bigclaw/operations.py`
  - Updated.
  - Reason: now owns the former reports dataclasses, renderers, writers,
    replay helpers, orchestration/takeover reporting helpers, and repo
    narrative export helpers.
- `src/bigclaw/__init__.py`
  - Updated.
  - Reason: package imports now source report symbols from operations and
    install a `bigclaw.reports` compatibility module backed by that surface.

### Follow-on tranche 27 inventory

- `src/bigclaw` Python files after tranche 27: `4`
- Repository-wide Python files after tranche 27: `52`
- Net repository-wide Python reduction: `56`
- `bigclaw-go` Go files after tranche 27: `267`
- Root `pyproject.toml`: absent
- Root `setup.py`: absent

### Follow-on tranche 27 validation

- `python3 -m py_compile src/bigclaw/operations.py src/bigclaw/__init__.py`
  - Result: success
- `PYTHONPATH=src python3 - <<'PY'` with `import bigclaw`, `import bigclaw.reports`, `import bigclaw.evaluation`, and `import bigclaw.run_detail`
  - Result: `import ok`; `bigclaw.reports.SharedViewContext.__module__ == "bigclaw.operations"`; `bigclaw.evaluation.BenchmarkSuiteResult.__module__ == "bigclaw.operations"`; `bigclaw.run_detail.RunDetailTab.__module__ == "bigclaw.operations"`
- `PYTHONPATH=src python3 -m pytest tests/test_reports.py tests/test_observability.py tests/test_evaluation.py`
  - Result: `48 passed in 0.08s`
- `PYTHONPATH=src python3 -m pytest tests/test_operations.py tests/test_control_center.py tests/test_repo_rollout.py`
  - Result: `25 passed in 0.06s`
- `PYTHONPATH=src python3 -m pytest tests/test_audit_events.py tests/test_runtime_matrix.py`
  - Result: `8 passed in 0.06s`
- `PYTHONPATH=src python3 -m pytest tests/test_planning.py tests/test_governance.py`
  - Result: `18 passed in 0.05s`
- `PYTHONPATH=src python3 -m pytest tests/test_parallel_validation_bundle.py`
  - Result: failed, unrelated pre-existing blocker in `bigclaw-go/scripts/e2e/export_validation_bundle.py` on Python 3.9: `TypeError: unsupported operand type(s) for |: 'type' and 'NoneType'` from `Path | None`
- `find src/bigclaw -type f -name '*.py' | wc -l`
  - Result: `4`
- `find . -type f -name '*.py' | wc -l`
  - Result: `52`
- `find bigclaw-go -type f -name '*.go' | wc -l`
  - Result: `267`
- `test -f pyproject.toml && echo present || echo absent`
  - Result: `absent`
- `test -f setup.py && echo present || echo absent`
  - Result: `absent`
- `git diff --check`
  - Result: success

### Follow-on tranche 28 queued

- Target `src/bigclaw/runtime.py` next.
- Reason: after the reports merge, the remaining runtime/worker surface still
  depends one-way on `observability.py`, while `operations.py` already consumes
  observability without any reverse edge, so the runtime layer can be absorbed
  into `src/bigclaw/observability.py` and preserved through a
  package-installed `bigclaw.runtime` compatibility shim.
- Planned edits:
  - Move the legacy runtime, queue, orchestration, scheduler, workflow,
    service, and governance helpers from `src/bigclaw/runtime.py` into
    `src/bigclaw/observability.py`.
  - Repoint package exports so `bigclaw.runtime` resolves to the
    observability-owned surface.
  - Delete `src/bigclaw/runtime.py`.

### Follow-on tranche 28

- `src/bigclaw/runtime.py`
  - Deleted.
  - Reason: the remaining legacy runtime/worker surface only depended on the
    observability layer, so it could be absorbed into
    `src/bigclaw/observability.py` while preserving `bigclaw.runtime` through a
    package-installed compatibility shim.
- `src/bigclaw/observability.py`
  - Updated.
  - Reason: now owns the former runtime, queue, orchestration, scheduler,
    workflow, service, and repo-governance helpers in addition to the existing
    models, event bus, ledger, and audit surface.
- `src/bigclaw/__init__.py`
  - Updated.
  - Reason: package imports now source runtime symbols from observability and
    install a `bigclaw.runtime` compatibility module backed by that surface.

### Follow-on tranche 28 inventory

- `src/bigclaw` Python files after tranche 28: `3`
- Repository-wide Python files after tranche 28: `51`
- Net repository-wide Python reduction: `57`
- `bigclaw-go` Go files after tranche 28: `267`
- Root `pyproject.toml`: absent
- Root `setup.py`: absent

### Follow-on tranche 28 validation

- `python3 -m py_compile src/bigclaw/observability.py src/bigclaw/operations.py src/bigclaw/__init__.py`
  - Result: success
- `PYTHONPATH=src python3 - <<'PY'` with `import bigclaw`, `import bigclaw.runtime`, and `import bigclaw.observability`
  - Result: `import ok`; `bigclaw.runtime.ClawWorkerRuntime.__module__ == "bigclaw.observability"`; `bigclaw.observability.ClawWorkerRuntime.__module__ == "bigclaw.observability"`; `hasattr(bigclaw.runtime, "LEGACY_MAINLINE_STATUS") == True`
- `PYTHONPATH=src python3 -m pytest tests/test_runtime_matrix.py tests/test_risk.py tests/test_event_bus.py tests/test_observability.py`
  - Result: `16 passed in 0.06s`
- `PYTHONPATH=src python3 -m pytest tests/test_reports.py tests/test_operations.py tests/test_control_center.py tests/test_repo_rollout.py`
  - Result: `59 passed in 0.09s`
- `PYTHONPATH=src python3 -m pytest tests/test_orchestration.py tests/test_scheduler.py tests/test_audit_events.py tests/test_planning.py tests/test_governance.py`
  - Result: `32 passed in 0.07s`
- `find src/bigclaw -type f -name '*.py' | wc -l`
  - Result: `3`
- `find . -type f -name '*.py' | wc -l`
  - Result: `51`
- `find bigclaw-go -type f -name '*.go' | wc -l`
  - Result: `267`
- `test -f pyproject.toml && echo present || echo absent`
  - Result: `absent`
- `test -f setup.py && echo present || echo absent`
  - Result: `absent`
- `git diff --check`
  - Result: success
- Known unrelated blocker retained from prior tranche:
  - `PYTHONPATH=src python3 -m pytest tests/test_parallel_validation_bundle.py`
  - Result: failed in `bigclaw-go/scripts/e2e/export_validation_bundle.py` on Python 3.9 because of `Path | None`

### Follow-on tranche 29 queued

- Target `src/bigclaw/observability.py` next.
- Reason: after the runtime merge, the remaining observability surface no
  longer had any reverse dependency from `operations.py` once the local imports
  were removed, so it could be absorbed into `src/bigclaw/operations.py` and
  preserved through an explicit `bigclaw.observability` package alias plus the
  existing `runtime`, `reports`, `models`, `event_bus`, `collaboration`, and
  `audit_events` compatibility modules.
- Planned edits:
  - Move the observability/event-bus/model surface from
    `src/bigclaw/observability.py` into `src/bigclaw/operations.py`.
  - Repoint package bootstrap so `bigclaw.observability` resolves to the
    operations-owned module.
  - Delete `src/bigclaw/observability.py`.

### Follow-on tranche 29

- `src/bigclaw/observability.py`
  - Deleted.
  - Reason: the remaining observability/event-bus/model surface could be
    absorbed into `src/bigclaw/operations.py` without leaving any reverse
    dependency edge, while package-level compatibility aliases preserve direct
    `bigclaw.observability` imports.
- `src/bigclaw/operations.py`
  - Updated.
  - Reason: now owns the former observability models, event bus, ledger,
    collaboration helpers, repo-sync telemetry, runtime surface, report
    surface, planning/governance surface, and operations surface.
- `src/bigclaw/__init__.py`
  - Updated.
  - Reason: package bootstrap now aliases `bigclaw.observability` directly to
    the merged operations module and sources top-level package exports from the
    same surface.

### Follow-on tranche 29 inventory

- `src/bigclaw` Python files after tranche 29: `2`
- Repository-wide Python files after tranche 29: `50`
- Net repository-wide Python reduction: `58`
- `bigclaw-go` Go files after tranche 29: `267`
- Root `pyproject.toml`: absent
- Root `setup.py`: absent

### Follow-on tranche 29 validation

- `python3 -m py_compile src/bigclaw/operations.py src/bigclaw/__init__.py`
  - Result: success
- `PYTHONPATH=src python3 - <<'PY'` with `import bigclaw`, `import bigclaw.observability`, `import bigclaw.runtime`, and `import bigclaw.reports`
  - Result: `import ok`; `bigclaw.observability.Task.__module__ == "bigclaw.operations"`; `bigclaw.runtime.ClawWorkerRuntime.__module__ == "bigclaw.operations"`; `bigclaw.reports.SharedViewContext.__module__ == "bigclaw.operations"`
- `PYTHONPATH=src python3 -m pytest tests/test_runtime_matrix.py tests/test_risk.py tests/test_event_bus.py tests/test_observability.py`
  - Result: `16 passed in 0.06s`
- `PYTHONPATH=src python3 -m pytest tests/test_reports.py tests/test_operations.py tests/test_control_center.py tests/test_repo_rollout.py tests/test_orchestration.py tests/test_scheduler.py tests/test_audit_events.py tests/test_planning.py tests/test_governance.py`
  - Result: `91 passed in 0.10s`
- `find src/bigclaw -type f -name '*.py' | wc -l`
  - Result: `2`
- `find . -type f -name '*.py' | wc -l`
  - Result: `50`
- `find bigclaw-go -type f -name '*.go' | wc -l`
  - Result: `267`
- `test -f pyproject.toml && echo present || echo absent`
  - Result: `absent`
- `test -f setup.py && echo present || echo absent`
  - Result: `absent`
- `git diff --check`
  - Result: success
- Known unrelated blocker retained from prior tranche:
  - `PYTHONPATH=src python3 -m pytest tests/test_parallel_validation_bundle.py`
  - Result: failed in `bigclaw-go/scripts/e2e/export_validation_bundle.py` on Python 3.9 because of `Path | None`
