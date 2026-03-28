# BIG-GO-923 pytest harness migration

## Scope

This issue migrates the current pytest bootstrap baseline toward a Go-native test harness for the `bigclaw-go` tree.

The Python-side harness surface in scope today is intentionally small:

- `tests/conftest.py`
- `tests/test_*.py` under the repository root

## Current Python and non-Go asset inventory

`tests/conftest.py` currently performs a single harness function:

- resolve the repository root from `tests/`
- prepend `<repo>/src` to `sys.path`
- make `from bigclaw...` imports work for pytest

Observed inventory at the time of migration:

- `56` Python test modules under `tests/`
- `47` modules directly importing `bigclaw...`
- `3` modules importing `pytest`: `test_audit_events.py`, `test_planning.py`, `test_roadmap.py`
- no shared pytest fixtures in `tests/` and no fixture definitions in `tests/conftest.py`
- `tests/conftest.py` does not import `pytest` and does not define pytest hooks; it is a plain import-path shim

This means the legacy pytest harness is an import bootstrap, not a fixture/runtime orchestration layer.

## Go replacement landed in this issue

The new Go-native baseline lives in `bigclaw-go/internal/testharness`.

It provides:

- `RepoRoot(tb)` to locate the `bigclaw-go` module root without relying on package cwd
- `ProjectRoot(tb)` to reach the parent repository root that still contains legacy `src/` and `tests/`
- `LegacySrcRoot(tb)` to resolve the legacy Python import root previously injected by `tests/conftest.py`
- `JoinRepoRoot(tb, elems...)` and `JoinProjectRoot(tb, elems...)` for stable fixture/report path resolution
- `ResolveProjectPath(tb, candidate)` for paths that may still be prefixed with `bigclaw-go/`
- `PrependPathEnv(tb, dir)` for path-based CLI bootstrapping
- `PrependPythonPathEnv(tb, dir)` and `BootstrapLegacyPythonPath(tb)` for Go-owned `PYTHONPATH` bootstrapping when tests still need Python runtime parity
- `PythonCommand(tb, args...)` for launching `python3` from the project root with legacy `src/` imports already bootstrapped
- `PytestCommand(tb, args...)` for launching `python3 -m pytest` through the same Go-owned bootstrap path while legacy pytest slices still exist
- `RequireExecutable(tb, name)` for shared skip-aware runtime probing when legacy Python tooling is still part of the migration boundary
- `PythonExecutable(tb)` for the canonical resolved Python runtime path used by adjacent Go migration tests
- `Chdir(tb, dir)` for temporary cwd changes with automatic cleanup
- `InventoryPytestAssets(tb)` to machine-check the remaining pytest surface (`56` test modules, `47` `bigclaw` importers, `3` `pytest` importers) instead of leaving that inventory only in prose
- `InventoryPytestAssets(tb)` now detects pytest usage via `import pytest`, `from pytest import ...`, and `pytest.` call sites so the `tests/conftest.py` deletion gate does not miss direct import forms
- `PytestAssetInventory.ConftestDeletionBlockers()` to keep the current `tests/conftest.py` removal blockers machine-checked from Go rather than only documented in markdown
- `PytestAssetInventory.CanDeleteConftest()` to expose the current deletion gate as a single Go-owned boolean for future migration slices
- `PytestAssetInventory.ConftestDeletionSummary()` to provide one stable, report-ready line for the current delete-readiness state
- `PytestAssetInventory.ConftestDeletionStatus()` to provide a structured status object for future CLI/report surfaces that need both the boolean gate and the blocker/count breakdown
- `bigclawctl pytest-harness [--json]` as a stable Go-owned command surface that prints the current pytest asset inventory, `tests/conftest.py` behavior flags, and structured deletion-gate status without relying on pytest itself
- `internal/legacyshim` tests now also assert that the frozen Python compile-check asset list still matches the checked-in `src/bigclaw/*.py` shim files that remain in scope for migration
- `internal/legacyshim` now runs a real checked-in `py_compile` pass against those shim files, so the remaining Python compatibility layer is regression-tested from Go without bespoke bootstrap code
- `internal/testharness` now includes a Python import smoke test that boots `PYTHONPATH` via the Go harness and imports `bigclaw.mapping` directly, proving the replacement covers the old `conftest.py` core responsibility
- `internal/testharness` now also proves `PytestCommand(...)` can run `tests/test_mapping.py` end-to-end through the Go bootstrap path, including when no `PYTHONPATH` is preconfigured, which becomes the reusable bridge while the remaining pytest slices are still being retired

First-batch adoption landed here:

- `internal/regression/*_test.go` now uses the shared repo-root baseline instead of ad hoc `../..` resolution and `runtime.Caller` plumbing
- `cmd/bigclawctl/migration_commands_test.go` now uses the shared cwd and `PATH` bootstrap helpers
- `cmd/bigclawctl/main_test.go` and `internal/refill/local_store_test.go` now use `testharness.Chdir(...)` instead of bespoke cwd save/restore code
- `cmd/bigclawctl/main_test.go` now also resolves the Python runtime via `testharness.RequireExecutable(...)` instead of hard-coding `python3`
- `cmd/bigclawctl/main_test.go` and `internal/legacyshim/compilecheck_test.go` now use `testharness.PythonExecutable(...)` as the shared Python runtime entry point

First migrated Python test slice now covered explicitly in Go:

- `tests/test_dashboard_run_contract.py`
  - `test_dashboard_run_contract_default_bundle_is_release_ready`
  - `test_dashboard_run_contract_audit_detects_missing_field_definitions_and_samples`
  - `test_dashboard_run_contract_round_trip_preserves_samples_and_audit`
  - covered by `bigclaw-go/internal/product/dashboard_run_contract_test.go`
- `tests/test_saved_views.py`
  - `test_saved_view_catalog_round_trip_preserves_manifest_shape`
  - `test_saved_view_catalog_audit_surfaces_configuration_gaps`
  - `test_saved_view_catalog_audit_round_trip_preserves_findings`
  - `test_render_saved_view_report_summarizes_views_and_digest_coverage`
  - covered by `bigclaw-go/internal/product/saved_views_test.go`
- `tests/test_legacy_shim.py`
  - `test_dev_smoke_shim_runs_without_pythonpath`
  - `test_create_issues_shim_help_runs_without_pythonpath`
  - `test_github_sync_shim_help_runs_without_pythonpath`
  - `test_workspace_bootstrap_shim_help_runs_without_pythonpath`
  - `test_symphony_workspace_bootstrap_shim_help_runs_without_pythonpath`
  - `test_symphony_workspace_validate_shim_help_runs_without_pythonpath`
  - `test_refill_shim_help_runs_without_pythonpath`
  - covered by `bigclaw-go/cmd/bigclawctl/migration_commands_test.go` and `bigclaw-go/cmd/bigclawctl/main_test.go`
- `tests/test_governance.py`
  - `test_scope_freeze_board_round_trip_preserves_manifest_shape`
  - `test_scope_freeze_audit_flags_backlog_governance_and_closeout_gaps`
  - `test_scope_freeze_audit_round_trip_and_ready_state`
  - `test_render_scope_freeze_report_summarizes_board_and_run_closeout_requirements`
  - covered by `bigclaw-go/internal/governance/freeze_test.go`
- `tests/test_workspace_bootstrap.py`
  - `test_repo_cache_key_derives_from_repo_locator`
  - `test_cache_root_for_repo_uses_repo_specific_directory`
  - `test_bootstrap_workspace_creates_shared_worktree_from_local_seed`
  - `test_second_workspace_reuses_warm_cache_without_full_clone`
  - `test_bootstrap_workspace_reuses_existing_issue_worktree`
  - `test_cleanup_workspace_preserves_shared_cache_for_future_reuse`
  - `test_bootstrap_recovers_from_stale_seed_directory_without_remote_reclone`
  - `test_cleanup_workspace_prunes_worktree_and_bootstrap_branch`
  - `test_validation_report_covers_three_workspaces_with_one_cache`
  - covered by `bigclaw-go/internal/bootstrap/bootstrap_test.go`
- `tests/test_control_center.py`
  - `test_queue_peek_tasks_returns_priority_order`
  - covered by `bigclaw-go/internal/queue/memory_queue_test.go`
  - `test_queue_control_center_summarizes_queue_and_execution_media`
  - covered by `bigclaw-go/internal/reporting/reporting_test.go` and `bigclaw-go/internal/api/server_test.go`
  - `test_queue_control_center_renders_shared_view_empty_state`
  - covered by `bigclaw-go/internal/reporting/reporting_test.go`
- `tests/test_validation_bundle_continuation_policy_gate.py`
  - `test_checked_in_policy_gate_matches_expected_shape`
  - covered by `bigclaw-go/internal/regression/validation_bundle_continuation_test.go`
- `tests/test_validation_bundle_continuation_scorecard.py`
  - `test_checked_in_continuation_scorecard_matches_expected_shape`
  - `test_continuation_scorecard_marks_lane_success_and_manual_boundary`
  - `test_continuation_scorecard_summarizes_recent_bundle_chain`
  - covered by `bigclaw-go/internal/regression/validation_bundle_continuation_test.go`
- `tests/test_live_shadow_bundle.py`
  - `test_checked_in_live_shadow_bundle_matches_expected_shape`
  - covered by `bigclaw-go/internal/regression/live_shadow_bundle_surface_test.go`
- `tests/test_live_shadow_scorecard.py`
  - `test_checked_in_live_shadow_scorecard_matches_expected_shape`
  - covered by `bigclaw-go/internal/regression/live_shadow_bundle_surface_test.go`
- `tests/test_cross_process_coordination_surface.py`
  - `test_checked_in_coordination_surface_matches_expected_shape`
  - covered by `bigclaw-go/internal/regression/coordination_contract_surface_test.go`
- `tests/test_subscriber_takeover_harness.py`
  - `test_checked_in_takeover_report_matches_local_harness_shape`
  - covered by `bigclaw-go/internal/regression/takeover_proof_surface_test.go`
- `tests/test_validation_policy.py`
  - `test_big602_validation_policy_blocks_issue_close_without_required_reports`
  - `test_big602_validation_policy_allows_issue_close_when_reports_complete`
  - covered by `bigclaw-go/internal/policy/validation_report_policy_test.go`
- `tests/test_repo_governance.py`
  - `test_repo_permission_matrix_resolves_roles`
  - `test_repo_audit_field_contract_is_deterministic`
  - covered by `bigclaw-go/internal/repo/governance_test.go`
- `tests/test_repo_registry.py`
  - `test_repo_registry_resolves_space_channel_and_agent_deterministically`
  - `test_repo_registry_round_trip`
  - covered by `bigclaw-go/internal/repo/repo_surfaces_test.go`
- `tests/test_repo_board.py`
  - `test_repo_board_create_reply_and_target_filtering`
  - covered by `bigclaw-go/internal/repo/repo_surfaces_test.go`
- `tests/test_repo_gateway.py`
  - `test_repo_gateway_normalization_and_audit_payload`
  - `test_repo_gateway_error_normalization_is_deterministic`
  - covered by `bigclaw-go/internal/repo/repo_surfaces_test.go`
- `tests/test_repo_triage.py`
  - `test_lineage_aware_recommendations`
  - `test_approval_evidence_packet_includes_candidate_and_accepted_hash`
  - covered by `bigclaw-go/internal/repo/repo_surfaces_test.go`
- `tests/test_mapping.py`
  - `test_map_priority`
  - `test_map_source_issue_to_task`
  - covered by `bigclaw-go/internal/intake/mapping_test.go`
- `tests/test_connectors.py`
  - `test_connectors_fetch_minimum_issue`
  - covered by `bigclaw-go/internal/intake/connector_test.go`
- `tests/test_risk.py`
  - `test_risk_scorer_keeps_simple_low_risk_work_low`
  - `test_risk_scorer_elevates_prod_browser_work`
  - `test_scheduler_uses_risk_score_to_require_approval`
  - covered by `bigclaw-go/internal/risk/risk_test.go` and `bigclaw-go/internal/scheduler/scheduler_test.go`
- `tests/test_queue.py`
  - `test_queue_persistence_and_priority`
  - `test_queue_creates_parent_directory_and_preserves_task_payload`
  - `test_queue_dead_letter_and_retry_persist_across_reload`
  - `test_queue_loads_legacy_list_storage`
  - covered by `bigclaw-go/internal/queue/file_queue_test.go`
- `tests/test_workflow.py`
  - `test_workpad_journal_can_replay_and_reload`
  - `test_acceptance_gate_rejects_missing_evidence`
  - `test_acceptance_gate_rejects_hold_pilot_scorecard`
  - covered by `bigclaw-go/internal/workflow/engine_test.go`
- `tests/test_models.py`
  - `test_flow_template_and_run_round_trip_preserve_steps_and_outputs`
  - covered by `bigclaw-go/internal/workflow/model_test.go`
  - `test_billing_summary_round_trip_preserves_rates_and_usage`
  - covered by `bigclaw-go/internal/billing/statement_test.go`
- `tests/test_github_sync.py`
  - `test_install_git_hooks_configures_core_hooks_path`
  - `test_ensure_repo_sync_pushes_head_to_origin`
  - `test_inspect_repo_sync_marks_dirty_worktree`
  - `test_ensure_repo_sync_fast_forwards_clean_branch_before_push`
  - `test_ensure_repo_sync_skips_pushing_clean_branch_at_origin_default_head`
  - covered by `bigclaw-go/internal/githubsync/sync_test.go`
- `tests/test_memory.py`
  - `test_big501_memory_store_reuses_history_and_injects_rules`
  - covered by `bigclaw-go/internal/memory/store_test.go`
- `tests/test_parallel_validation_bundle.py`
  - `test_export_validation_bundle_generates_latest_reports_and_index`
  - checked-in report/index contract covered by
    `bigclaw-go/internal/regression/live_validation_index_test.go`,
    `bigclaw-go/internal/regression/live_validation_index_summary_test.go`,
    `bigclaw-go/internal/regression/live_validation_index_markdown_test.go`,
    `bigclaw-go/internal/regression/runtime_report_followup_docs_test.go`,
    and `bigclaw-go/internal/regression/shared_queue_companion_summary_test.go`

Still legacy-only for continuation policy tooling:

- Python script execution semantics in `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py`
- Python script execution semantics in `bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py`
- The checked-in report shapes are now guarded by Go regression tests, but direct script-runtime parity remains on the Python side

Still legacy-only for live-shadow bundle tooling:

- Python script execution semantics in `bigclaw-go/scripts/migration/export_live_shadow_bundle.py`
- Python script execution semantics in `bigclaw-go/scripts/migration/live_shadow_scorecard.py`
- The checked-in bundle and scorecard report shapes are guarded by Go regression tests, but the script runtime paths remain Python-owned

Still legacy-only for coordination/takeover tooling:

- Python script execution semantics in `bigclaw-go/scripts/e2e/cross_process_coordination_surface.py`
- Python script execution semantics in `bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py`
- The checked-in coordination and takeover report contracts are guarded by Go regression tests, but the script runtime paths remain Python-owned

Still legacy-only within `tests/test_legacy_shim.py`:

- Python wrapper argument translation helpers in `src/bigclaw/legacy_shim.py`
- These are compatibility shims rather than target Go mainline behavior and can be retired only when the Python wrappers themselves are removed from supported validation paths

Still legacy-only for bundle export runtime semantics:

- Python script execution semantics in `bigclaw-go/scripts/e2e/export_validation_bundle.py`
- Go now guards the checked-in live-validation summary/index, shared-queue companion pointers, and continuation-gate/index markdown surface, but direct script runtime parity remains on the Python side

Still partially migrated for repo run-link runtime semantics:

- `tests/test_repo_links.py` now overlaps with Go coverage in `bigclaw-go/internal/repo/repo_surfaces_test.go`, `bigclaw-go/internal/triage/repo_test.go`, and `bigclaw-go/internal/api/server_test.go`
- The remaining Python-owned piece is the legacy `TaskRun.record_closeout()` / `TaskRun.from_dict()` runtime path in `src/bigclaw/observability.py`, which has not been replaced by a single Go-owned run model in this issue

Still partially migrated for workflow/event persistence semantics:

- `tests/test_workflow.py` overlaps with Go coverage in `bigclaw-go/internal/workflow/model_test.go`, `bigclaw-go/internal/workflow/engine_test.go`, `bigclaw-go/internal/workflow/closeout_test.go`, and `bigclaw-go/internal/workflow/orchestration_test.go`
- The remaining Python-owned pieces are ledger-backed workflow runtime/report paths such as repo-sync audit reporting and observability persistence, which do not yet map to one Go-owned runtime package in this issue

Still partially migrated for scheduler/runtime execution semantics:

- `tests/test_scheduler.py` overlaps with Go routing and policy coverage in `bigclaw-go/internal/scheduler/scheduler_test.go`
- `tests/test_runtime.py` overlaps with Go worker acceptance and execution-gate coverage in `bigclaw-go/internal/workflow/engine_test.go` and `bigclaw-go/internal/worker/runtime_test.go`
- `tests/test_execution_flow.py` overlaps with Go scheduler and reporting coverage in `bigclaw-go/internal/scheduler/scheduler_test.go`, `bigclaw-go/internal/worker/runtime_test.go`, and `bigclaw-go/internal/reporting/reporting_test.go`
- The remaining Python-owned pieces are the legacy `Scheduler.execute()` / `ObservabilityLedger` runtime path, Python sandbox-router/tool-runtime objects in `src/bigclaw/runtime.py`, and markdown/html task-run report export behavior in `src/bigclaw/reports.py`

Still partially migrated for orchestration commercialization semantics:

- `tests/test_orchestration.py` overlaps with Go coverage in `bigclaw-go/internal/workflow/orchestration_test.go` and scheduler assessment coverage in `bigclaw-go/internal/scheduler/scheduler_test.go`
- The remaining Python-owned pieces are orchestration-plan rendering into legacy run-ledger/report artifacts and commercialization-report surfaces still implemented in `src/bigclaw/orchestration.py`, `src/bigclaw/observability.py`, and `src/bigclaw/reports.py`

Still partially migrated for broader model-runtime semantics:

- `tests/test_models.py` now overlaps with Go coverage in `bigclaw-go/internal/workflow/model_test.go` and `bigclaw-go/internal/billing/statement_test.go`
- The remaining Python-owned pieces are richer risk-assessment and triage record model surfaces that do not have a single equivalent Go model package in this issue

Still legacy-only for planning, roadmap, and deprecation metadata surfaces:

- `tests/test_planning.py`, `tests/test_roadmap.py`, and `tests/test_deprecation.py` still exercise Python-owned modules in `src/bigclaw/planning.py`, `src/bigclaw/roadmap.py`, and `src/bigclaw/deprecation.py`
- These files mainly validate manifest/report rendering, baseline gating copy, and deprecation-warning metadata rather than shared pytest bootstrap behavior, so they remain outside this issue's Go harness replacement slice

Still partially migrated for observability and event-bus runtime semantics:

- `tests/test_event_bus.py` overlaps only at the Go event-stream primitive level in `bigclaw-go/internal/events/bus_test.go`
- `tests/test_observability.py` overlaps only at the Go recorder/audit primitive level in `bigclaw-go/internal/observability/recorder_test.go`, `bigclaw-go/internal/observability/audit_test.go`, and `bigclaw-go/internal/observability/audit_spec_test.go`
- The remaining Python-owned pieces are `TaskRun`, `ObservabilityLedger`, collaboration-thread synthesis, and run-detail/report rendering in `src/bigclaw/observability.py` and `src/bigclaw/reports.py`; these have no single Go-owned runtime replacement in this issue

Still partially migrated for broader reporting/studio semantics:

- `tests/test_reports.py` overlaps with Go reporting coverage in `bigclaw-go/internal/reporting/reporting_test.go` for weekly operations bundles, queue control center rendering, engineering overview bundles, and dashboard/policy-center reporting
- The remaining Python-owned pieces are report-studio narratives, pilot scorecards, launch/final-delivery checklists, billing-entitlements pages, orchestration canvases/portfolios, takeover queue synthesis, and task-run report/detail rendering in `src/bigclaw/reports.py`

## Migration plan

1. Treat `internal/testharness` as the only shared bootstrap layer for Go tests that need repository-relative assets or CLI environment setup.
2. Continue porting Python contract/report tests into `bigclaw-go/internal/...` packages on top of that harness instead of extending pytest infrastructure.
3. Keep Python tests runnable only as long as there are remaining `src/bigclaw` behaviors without Go coverage.

Recommended next migration slices:

- `tests/test_dashboard_run_contract.py` into `bigclaw-go/internal/product`
- `tests/test_saved_views.py` into `bigclaw-go/internal/product`
- `tests/test_legacy_shim.py` into `bigclaw-go/internal/legacyshim` and `cmd/bigclawctl`
- `tests/test_workspace_bootstrap.py` into `bigclaw-go/internal/bootstrap`
- broader runtime/ledger-backed workflow and queue surfaces that still depend on Python-owned persistence models

## Deletion gate for legacy Python harness

`tests/conftest.py` is safe to delete only when all of the following are true:

- no remaining validation lane depends on `python3 -m pytest`
- no remaining test module imports `bigclaw...` from `src/`
- Go replacements cover the active regression surface for the remaining Python tests
- a repo-wide validation run succeeds without Python path injection

Current machine-checked blockers in this issue are:

- `56 legacy pytest modules remain under tests/`
- `47 legacy pytest modules still import bigclaw from src/`
- `3 legacy pytest modules still import pytest directly`

The `pytest` blocker count is computed from Go-owned inventory code and now covers all three currently supported detection forms:

- `import pytest`
- `from pytest import ...`
- `pytest.<member>`

Current machine-checked single-line summary is:

- `conftest_delete_ready=false blockers=56 legacy pytest modules remain under tests/; 47 legacy pytest modules still import bigclaw from src/; 3 legacy pytest modules still import pytest directly`

Current Go-owned command surface for this state:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go run ./cmd/bigclawctl pytest-harness --project-root .. --json
```

That command emits:

- the current inventory summary/counts
- the remaining `tests/test_*.py` modules, `bigclaw` importers, and `pytest` importers
- the current `tests/conftest.py` behavior flags
- the structured `conftest` deletion gate used by the migration report and tests

Until then, `tests/conftest.py` remains a compatibility shim and should not grow new behavior.

## Regression commands

Primary validation for this issue:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923 && python3 -m pytest tests/test_mapping.py -q
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/testharness
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./cmd/bigclawctl
```

Observed results for this issue:

- `python3 -m pytest tests/test_mapping.py -q` passed (`.. [100%]`) on the latest issue branch state, after the current `conftest` deletion-gate assertions landed, confirming the current `tests/conftest.py` import bootstrap still supports legacy `src/bigclaw` imports.
- `go test ./internal/testharness` passed on the latest issue branch state, including the Go-side Python import smoke for `bigclaw.mapping`, the Go-launched legacy pytest smoke, the machine-checked `conftest` deletion gate, and the direct-import detection coverage for `import pytest`, `from pytest import ...`, and `pytest.<member>`, confirming the replacement helpers and deletion-gate logic are stable.
- `go test ./cmd/bigclawctl` passed on the latest issue branch state, including the new `pytest-harness` command surface that exposes the inventory summary and structured `conftest` deletion-gate status from Go-owned code.

Deletion-readiness validation for the legacy Python harness, once migration is further along:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923 && python3 -m pytest tests
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./...
```
