# BIG-GO-923 pytest harness migration

## Scope

This issue migrates the current pytest bootstrap baseline toward a Go-native test harness for the `bigclaw-go` tree.

The Python-side harness surface in scope today is intentionally small:

- `tests/conftest.py`
- `tests/test_*.py` under the repository root
- `pyproject.toml` pytest dependency/config stanzas that still define the legacy pytest lane

## Current Python and non-Go asset inventory

`tests/conftest.py` previously performed a single harness function:

- resolve the repository root from `tests/`
- prepend `<repo>/src` to `sys.path`
- make `from bigclaw...` imports work for pytest

Observed inventory at the current branch state:

- `28` Python test modules under `tests/`
- `28` modules directly importing `bigclaw...`
- `0` modules now import `pytest` directly within `tests/`
- `pyproject.toml` no longer declares `pytest` in the default `dev` extra
- `pyproject.toml` no longer defines `[tool.pytest.ini_options]`; legacy Python tests are now treated as migration-only manual lanes rather than the default repo test baseline
- `0` active `src/`/`tests/` Python files now embed explicit pytest validation commands after this issue's Go-first command cleanup
- no shared pytest fixtures in `tests/` and no fixture definitions were ever present in `tests/conftest.py`
- the deleted `tests/conftest.py` did not import `pytest` and did not define pytest hooks; it was only an import-path shim

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
- `PythonCommandAt(projectRoot, pythonBin, args...)` and `PytestCommandAt(projectRoot, pythonBin, args...)` for non-test Go surfaces such as `bigclawctl` that still need to run migration-only Python checks without re-implementing legacy `PYTHONPATH` bootstrap rules
- `RequireExecutable(tb, name)` for shared skip-aware runtime probing when legacy Python tooling is still part of the migration boundary
- `PythonExecutable(tb)` for the canonical resolved Python runtime path used by adjacent Go migration tests
- `Chdir(tb, dir)` for temporary cwd changes with automatic cleanup
- `InventoryPytestAssets(tb)` to machine-check the remaining pytest surface (`28` test modules, `28` `bigclaw` importers, `0` direct `pytest` importers) instead of leaving that inventory only in prose
- `InventoryPytestAssets(tb)` now walks `tests/` recursively, so legacy pytest files moved into nested subdirectories cannot silently escape the Go-owned inventory gate
- `InventoryPytestAssets(tb)` now detects pytest usage via `import pytest`, `from pytest import ...`, and `pytest.` call sites so the `tests/conftest.py` deletion gate does not miss direct import forms
- `InventoryPytestAssets(tb)` now also machine-checks whether `pyproject.toml` still declares pytest as a dev dependency or still defines `[tool.pytest.ini_options]`, so the remaining non-Go pytest infrastructure is tracked in the same report as `tests/conftest.py`
- `InventoryPytestAssets(tb)` now also machine-checks active `python3 -m pytest` command references inside `src/` and `tests/`, so the delete gate covers validation-lane command strings and not just imported test files/config; current issue state has `0` such active refs
- `PytestAssetInventory.ConftestDeletionBlockers()` to keep the current `tests/conftest.py` removal blockers machine-checked from Go rather than only documented in markdown
- `PytestAssetInventory.CanDeleteConftest()` to expose the current deletion gate as a single Go-owned boolean for future migration slices
- `PytestAssetInventory.ConftestDeletionSummary()` to provide one stable, report-ready line for the current delete-readiness state
- `PytestAssetInventory.ConftestDeletionStatus()` to provide a structured status object for future CLI/report surfaces that need both the boolean gate and the blocker/count breakdown
- `bigclawctl pytest-harness [--json]` as a stable Go-owned command surface that prints the current pytest asset inventory, `tests/conftest.py` presence/behavior flags, and structured deletion-gate status without relying on pytest itself
- `bigclawctl legacy-python pytest --repo .. --python python3 -- -- <pytest-args...>` as the Go-owned runtime wrapper for the remaining migration-only Python test slices after deleting `tests/conftest.py` and removing pytest from the default repo baseline
- `bigclaw-go/docs/reports/pytest-harness-status.json` as the checked-in machine-generated snapshot of the current pytest/conftest migration boundary
- `internal/regression/pytest_harness_status_test.go` to fail Go regression if the checked-in snapshot drifts from the live repo inventory or deletion gate
- `internal/legacyshim` tests now also assert that the frozen Python compile-check asset list still matches the checked-in `src/bigclaw/*.py` shim files that remain in scope for migration
- `internal/legacyshim` now runs a real checked-in `py_compile` pass against those shim files, so the remaining Python compatibility layer is regression-tested from Go without bespoke bootstrap code
- `internal/testharness` now includes a Python import smoke test that boots `PYTHONPATH` via the Go harness and imports `bigclaw.mapping` directly, proving the replacement covers the old `conftest.py` core responsibility
- `internal/testharness` now also proves `PytestCommand(...)` can run a temporary pytest module end-to-end through the Go bootstrap path, including when no `PYTHONPATH` is preconfigured, which becomes the reusable bridge while the remaining pytest slices are still being retired

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
  - retired in this issue; coverage lives in `bigclaw-go/internal/product/dashboard_run_contract_test.go`
- `tests/test_saved_views.py`
  - `test_saved_view_catalog_round_trip_preserves_manifest_shape`
  - `test_saved_view_catalog_audit_surfaces_configuration_gaps`
  - `test_saved_view_catalog_audit_round_trip_preserves_findings`
  - `test_render_saved_view_report_summarizes_views_and_digest_coverage`
  - retired in this issue; coverage lives in `bigclaw-go/internal/product/saved_views_test.go`
- `tests/test_legacy_shim.py`
  - `test_dev_smoke_shim_runs_without_pythonpath`
  - `test_create_issues_shim_help_runs_without_pythonpath`
  - `test_github_sync_shim_help_runs_without_pythonpath`
  - `test_workspace_bootstrap_shim_help_runs_without_pythonpath`
  - `test_symphony_workspace_bootstrap_shim_help_runs_without_pythonpath`
  - `test_symphony_workspace_validate_shim_help_runs_without_pythonpath`
  - `test_refill_shim_help_runs_without_pythonpath`
  - retired in this issue; coverage lives in `bigclaw-go/internal/legacyshim/python_wrappers_test.go`
- `tests/test_parallel_refill.py`
  - `test_parallel_refill_queue_records_unique_identifiers`
  - `test_parallel_refill_queue_selects_first_runnable_draft_slices`
  - retired in this issue; coverage lives in `bigclaw-go/internal/refill/queue_test.go`
- `tests/test_repo_collaboration.py`
  - `test_merge_collaboration_threads_combines_native_and_repo_surfaces`
  - retired in this issue; coverage lives in `bigclaw-go/internal/repo/repo_surfaces_test.go`
- `tests/test_pilot.py`
  - `test_big701_pilot_ready_when_kpis_pass_and_no_incidents`
  - `test_big701_render_pilot_report_contains_readiness_fields`
  - retired in this issue; coverage lives in `bigclaw-go/internal/product/pilot_test.go`
- `tests/test_cost_control.py`
  - `test_big503_cost_controller_degrades_when_high_medium_over_budget`
  - `test_big503_cost_controller_pauses_when_even_docker_exceeds_budget`
  - `test_big503_cost_controller_respects_budget_override_amount`
  - retired in this issue; coverage lives in `bigclaw-go/internal/scheduler/cost_control_test.go`
- `tests/test_models.py`
  - `test_risk_assessment_round_trip_preserves_signals_and_mitigations`
  - `test_triage_record_round_trip_preserves_queue_labels_and_actions`
  - `test_flow_template_and_run_round_trip_preserve_steps_and_outputs`
  - `test_billing_summary_round_trip_preserves_rates_and_usage`
  - retired in this issue; coverage lives in `bigclaw-go/internal/risk/assessment_test.go`, `bigclaw-go/internal/triage/record_test.go`, `bigclaw-go/internal/workflow/model_test.go`, and `bigclaw-go/internal/billing/statement_test.go`
- `tests/test_runtime.py`
  - `test_sandbox_router_maps_execution_media`
  - `test_tool_runtime_blocks_disallowed_tool_and_audits`
  - `test_worker_runtime_returns_tool_results_for_approved_task`
  - `test_scheduler_records_worker_runtime_results_and_waits_on_high_risk`
  - `test_scheduler_pauses_execution_when_budget_cannot_cover_docker`
  - retired in this issue; coverage lives in `bigclaw-go/internal/worker/runtime_test.go` and `bigclaw-go/internal/workflow/engine_test.go`
- `tests/test_runtime_matrix.py`
  - `test_big301_worker_lifecycle_is_stable_with_multiple_tools`
  - `test_big302_risk_routes_to_expected_sandbox_mediums`
  - `test_big303_tool_runtime_policy_and_audit_chain`
  - retired in this issue; coverage lives in `bigclaw-go/internal/worker/runtime_test.go` and `bigclaw-go/internal/scheduler/scheduler_test.go`
- `tests/test_audit_events.py`
  - `test_p0_audit_event_specs_define_required_operational_events`
  - `test_task_run_audit_spec_event_requires_required_fields`
  - `test_scheduler_emits_p0_operational_audit_events`
  - `test_workflow_records_canonical_approval_event`
  - `test_reports_accept_canonical_handoff_and_takeover_events`
  - retired in this issue; coverage lives in `bigclaw-go/internal/observability/audit_spec_test.go`, `bigclaw-go/internal/observability/audit_test.go`, `bigclaw-go/internal/observability/recorder_test.go`, `bigclaw-go/internal/worker/runtime_test.go`, and `bigclaw-go/internal/workflow/orchestration_test.go`
- `tests/test_issue_archive.py`
  - `test_issue_priority_archive_round_trip_preserves_manifest_shape`
  - `test_issue_priority_archive_audit_flags_owner_priority_category_and_open_p0_gaps`
  - `test_issue_priority_archive_audit_round_trip_and_ready_state`
  - `test_render_issue_priority_archive_report_summarizes_findings_and_rollups`
  - retired in this issue; coverage lives in `bigclaw-go/internal/product/issue_archive_test.go`
- `tests/test_governance.py`
  - `test_scope_freeze_board_round_trip_preserves_manifest_shape`
  - `test_scope_freeze_audit_flags_backlog_governance_and_closeout_gaps`
  - `test_scope_freeze_audit_round_trip_and_ready_state`
  - `test_render_scope_freeze_report_summarizes_board_and_run_closeout_requirements`
  - retired in this issue; coverage lives in `bigclaw-go/internal/governance/freeze_test.go`
- `tests/test_roadmap.py`
  - `test_build_execution_pack_roadmap_maps_epics_to_phases`
  - `test_execution_pack_roadmap_rejects_duplicate_owners`
  - covered by `bigclaw-go/internal/product/roadmap_test.go`
- `tests/test_deprecation.py`
  - `test_warn_legacy_runtime_surface_emits_deprecation_warning`
  - `test_legacy_runtime_modules_expose_go_mainline_replacements`
  - `test_legacy_service_surface_emits_go_first_warning`
  - `test_service_module_exposes_go_mainline_replacement`
  - covered by `bigclaw-go/internal/legacyshim/deprecation_test.go`
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
  - retired in this issue; coverage lives in `bigclaw-go/internal/bootstrap/bootstrap_test.go`
- `tests/test_control_center.py`
  - `test_queue_peek_tasks_returns_priority_order`
  - retired in this issue; coverage lives in `bigclaw-go/internal/queue/memory_queue_test.go`
  - `test_queue_control_center_summarizes_queue_and_execution_media`
  - retired in this issue; coverage lives in `bigclaw-go/internal/reporting/reporting_test.go` and `bigclaw-go/internal/api/server_test.go`
  - `test_queue_control_center_renders_shared_view_empty_state`
  - retired in this issue; coverage lives in `bigclaw-go/internal/reporting/reporting_test.go`
- `tests/test_validation_bundle_continuation_policy_gate.py`
  - `test_checked_in_policy_gate_matches_expected_shape`
  - retired in this issue; checked-in report shape remains covered by `bigclaw-go/internal/regression/validation_bundle_continuation_test.go`, and script/runtime behavior is covered by `bigclaw-go/internal/regression/validation_bundle_continuation_policy_runtime_test.go`
- `tests/test_validation_bundle_continuation_scorecard.py`
  - `test_checked_in_continuation_scorecard_matches_expected_shape`
  - `test_continuation_scorecard_marks_lane_success_and_manual_boundary`
  - `test_continuation_scorecard_summarizes_recent_bundle_chain`
  - retired in this issue; checked-in report shape remains covered by `bigclaw-go/internal/regression/validation_bundle_continuation_test.go`, and script/runtime behavior is covered by `bigclaw-go/internal/regression/validation_bundle_continuation_scorecard_runtime_test.go`
- `tests/test_live_shadow_bundle.py`
  - `test_checked_in_live_shadow_bundle_matches_expected_shape`
  - retired in this issue; checked-in report shape remains covered by `bigclaw-go/internal/regression/live_shadow_bundle_surface_test.go`, and script/runtime behavior is covered by `bigclaw-go/internal/regression/live_shadow_bundle_runtime_test.go`
- `tests/test_live_shadow_scorecard.py`
  - `test_checked_in_live_shadow_scorecard_matches_expected_shape`
  - retired in this issue; checked-in report shape remains covered by `bigclaw-go/internal/regression/live_shadow_bundle_surface_test.go`, and script/runtime behavior is covered by `bigclaw-go/internal/regression/live_shadow_scorecard_runtime_test.go`
- `tests/test_cross_process_coordination_surface.py`
  - `test_checked_in_coordination_surface_matches_expected_shape`
  - retired in this issue; checked-in report shape remains covered by `bigclaw-go/internal/regression/coordination_contract_surface_test.go`, and script/runtime behavior is covered by `bigclaw-go/internal/regression/cross_process_coordination_surface_runtime_test.go`
- `tests/test_subscriber_takeover_harness.py`
  - `test_checked_in_takeover_report_matches_local_harness_shape`
  - retired in this issue; checked-in report shape remains covered by `bigclaw-go/internal/regression/takeover_proof_surface_test.go`, and script/runtime behavior is covered by `bigclaw-go/internal/regression/subscriber_takeover_fault_matrix_runtime_test.go`
- `tests/test_shadow_matrix_corpus.py`
  - `test_shadow_matrix_report_records_corpus_coverage_scorecard`
  - retired in this issue; checked-in report shape remains covered by `bigclaw-go/internal/regression/production_corpus_surface_test.go`, and script/runtime behavior is covered by `bigclaw-go/internal/regression/shadow_matrix_runtime_test.go`
- `tests/test_followup_digests.py`
  - `test_followup_digests_capture_links_and_constraints`
  - `test_followup_indexes_reference_new_digests`
  - retired in this issue; coverage lives in `bigclaw-go/internal/regression/followup_digests_test.go`
- `tests/test_validation_policy.py`
  - `test_big602_validation_policy_blocks_issue_close_without_required_reports`
  - `test_big602_validation_policy_allows_issue_close_when_reports_complete`
  - retired in this issue; coverage lives in `bigclaw-go/internal/policy/validation_report_policy_test.go`
- `tests/test_repo_governance.py`
  - `test_repo_permission_matrix_resolves_roles`
  - `test_repo_audit_field_contract_is_deterministic`
  - retired in this issue; coverage lives in `bigclaw-go/internal/repo/governance_test.go`
- `tests/test_repo_registry.py`
  - `test_repo_registry_resolves_space_channel_and_agent_deterministically`
  - `test_repo_registry_round_trip`
  - retired in this issue; coverage lives in `bigclaw-go/internal/repo/repo_surfaces_test.go`
- `tests/test_repo_board.py`
  - `test_repo_board_create_reply_and_target_filtering`
  - retired in this issue; coverage lives in `bigclaw-go/internal/repo/repo_surfaces_test.go`
- `tests/test_repo_gateway.py`
  - `test_repo_gateway_normalization_and_audit_payload`
  - `test_repo_gateway_error_normalization_is_deterministic`
  - retired in this issue; coverage lives in `bigclaw-go/internal/repo/repo_surfaces_test.go`
- `tests/test_repo_triage.py`
  - `test_lineage_aware_recommendations`
  - `test_approval_evidence_packet_includes_candidate_and_accepted_hash`
  - retired in this issue; coverage lives in `bigclaw-go/internal/repo/repo_surfaces_test.go`
- `tests/test_connectors.py`
  - `test_connectors_fetch_minimum_issue`
  - covered by `bigclaw-go/internal/intake/connector_test.go`
- `tests/test_risk.py`
  - `test_risk_scorer_keeps_simple_low_risk_work_low`
  - `test_risk_scorer_elevates_prod_browser_work`
  - `test_scheduler_uses_risk_score_to_require_approval`
  - retired in this issue; coverage lives in `bigclaw-go/internal/risk/risk_test.go` and `bigclaw-go/internal/scheduler/scheduler_test.go`
- `tests/test_queue.py`
  - `test_queue_persistence_and_priority`
  - `test_queue_creates_parent_directory_and_preserves_task_payload`
  - `test_queue_dead_letter_and_retry_persist_across_reload`
  - `test_queue_loads_legacy_list_storage`
  - retired in this issue; coverage lives in `bigclaw-go/internal/queue/file_queue_test.go`
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
  - retired in this issue; coverage lives in `bigclaw-go/internal/githubsync/sync_test.go`
- `tests/test_memory.py`
  - `test_big501_memory_store_reuses_history_and_injects_rules`
  - retired in this issue; coverage lives in `bigclaw-go/internal/memory/store_test.go`
- `tests/test_parallel_validation_bundle.py`
  - `test_export_validation_bundle_generates_latest_reports_and_index`
  - retired in this issue; checked-in report/index contract remains covered by
    `bigclaw-go/internal/regression/live_validation_index_test.go`,
    `bigclaw-go/internal/regression/live_validation_index_summary_test.go`,
    `bigclaw-go/internal/regression/live_validation_index_markdown_test.go`,
    `bigclaw-go/internal/regression/runtime_report_followup_docs_test.go`,
    and `bigclaw-go/internal/regression/shared_queue_companion_summary_test.go`; script/runtime behavior is covered by `bigclaw-go/internal/regression/live_validation_bundle_runtime_test.go`

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

`tests/test_legacy_shim.py` is now retired from the legacy pytest lane:

- its wrapper argument translation and shim entrypoint checks are now covered by `bigclaw-go/internal/legacyshim/python_wrappers_test.go`
- the underlying Python compatibility module `src/bigclaw/legacy_shim.py` still exists for migration-time wrapper support, but its regression coverage is now Go-owned

`tests/test_parallel_refill.py` is now retired from the legacy pytest lane:

- its checked-in queue accessors and candidate-selection contract now live under `bigclaw-go/internal/refill/queue_test.go`
- the underlying queue source of truth remains `docs/parallel-refill-queue.json`, but its regression coverage is now Go-owned

`tests/test_repo_collaboration.py` is now retired from the legacy pytest lane:

- its collaboration-thread merge behavior now lives under `bigclaw-go/internal/repo/collaboration.go` and `bigclaw-go/internal/repo/repo_surfaces_test.go`
- the repo discussion board and collaboration-thread merge contract are now covered from the Go mainline

`tests/test_pilot.py` is now retired from the legacy pytest lane:

- its pilot KPI pass-rate, readiness, and report-heading contract now lives under `bigclaw-go/internal/product/pilot.go` and `bigclaw-go/internal/product/pilot_test.go`
- broader report-studio pilot scorecard surfaces in `src/bigclaw/reports.py` still remain outside this issue slice

`tests/test_cost_control.py` is now retired from the legacy pytest lane:

- its budget estimate/degrade/pause contract now lives under `bigclaw-go/internal/scheduler/cost_control.go` and `bigclaw-go/internal/scheduler/cost_control_test.go`
- scheduler-wide routing policy still remains in `bigclaw-go/internal/scheduler/scheduler.go`, but the standalone cost-controller behavior is now Go-owned

`tests/test_models.py` is now retired from the legacy pytest lane:

- its risk-assessment JSON contract now lives under `bigclaw-go/internal/risk/assessment_test.go`
- its triage-record JSON contract now lives under `bigclaw-go/internal/triage/record_test.go`
- its workflow template/run JSON contract now lives under `bigclaw-go/internal/workflow/model_test.go`
- its billing summary JSON contract now lives under `bigclaw-go/internal/billing/statement_test.go`

`tests/test_runtime.py` is now retired from the legacy pytest lane:

- its worker acceptance and execution-gate contract now lives under `bigclaw-go/internal/workflow/engine_test.go`
- its runtime processing, blocked execution, and lifecycle event coverage now lives under `bigclaw-go/internal/worker/runtime_test.go`

`tests/test_runtime_matrix.py` is now retired from the legacy pytest lane:

- its multi-tool worker lifecycle and policy-audit coverage now lives under `bigclaw-go/internal/worker/runtime_test.go`
- its sandbox medium routing coverage now lives under `bigclaw-go/internal/scheduler/scheduler_test.go`

`tests/test_audit_events.py` is now retired from the legacy pytest lane:

- its required operational audit-spec contract now lives under `bigclaw-go/internal/observability/audit_spec_test.go` and `bigclaw-go/internal/observability/audit_test.go`
- its canonical approval-event recording coverage now lives under `bigclaw-go/internal/observability/recorder_test.go`
- its handoff/takeover event-payload coverage now lives under `bigclaw-go/internal/worker/runtime_test.go` and `bigclaw-go/internal/workflow/orchestration_test.go`

`tests/test_issue_archive.py` is now retired from the legacy pytest lane:

- its issue-priority archive JSON contract, audit rollups, and report rendering now live under `bigclaw-go/internal/product/issue_archive.go` and `bigclaw-go/internal/product/issue_archive_test.go`

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

Still legacy-only for planning metadata surfaces:

- `tests/test_planning.py` still exercises Python-owned modules in `src/bigclaw/planning.py`
- This file mainly validates manifest/report rendering and baseline gating copy rather than shared pytest bootstrap behavior, so it remains outside this issue's Go harness replacement slice

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

- `tests/test_workspace_bootstrap.py` into `bigclaw-go/internal/bootstrap`
- broader runtime/ledger-backed workflow and queue surfaces that still depend on Python-owned persistence models

## Deletion gate for legacy Python harness

`tests/conftest.py` is now deleted in this issue because all of the following are true:

- no remaining supported validation lane depends on implicit pytest path bootstrapping
- `pyproject.toml` no longer declares pytest as a supported default test dependency and no longer defines `[tool.pytest.ini_options]`
- Go-owned `PythonCommand(...)` / `PytestCommand(...)` and explicit `PYTHONPATH=src` commands cover the remaining migration-only manual lanes
- `tests/conftest.py` did not import `pytest`, define fixtures/hooks, or declare `pytest_plugins`
- Go replacements cover the active regression surface for the remaining Python tests
- a repo-wide validation run succeeds without Python path injection

Current machine-checked blockers in this issue are:

- `18 legacy pytest modules remain under tests/`
- `18 legacy pytest modules still import bigclaw from src/`

The `pytest` blocker count is computed from Go-owned inventory code and now covers all three currently supported detection forms:

- `import pytest`
- `from pytest import ...`
- `pytest.<member>`

Current machine-checked single-line summary is:

- `conftest_delete_ready=true blockers=none`
- `legacy_pytest_delete_ready=false blockers=18 legacy pytest modules remain under tests/; 18 legacy pytest modules still import bigclaw from src/`

Current Go-owned command surface for this state:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go run ./cmd/bigclawctl pytest-harness --project-root .. --json
```

Checked-in machine-readable snapshot for the current branch state:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go run ./cmd/bigclawctl pytest-harness --project-root .. --report-path docs/reports/pytest-harness-status.json --json
```

That command emits:

- the current inventory summary/counts
- the current `pyproject.toml` pytest dependency/config flags, including whether the default repo baseline has already stopped declaring pytest
- the active `src/`/`tests/` files that still embed explicit pytest validation commands
- the remaining `tests/test_*.py` modules, `bigclaw` importers, and `pytest` importers
- whether `tests/conftest.py` still exists plus any remaining behavior flags if it does
- the structured `conftest` deletion gate used by the migration report and tests
- the structured legacy pytest asset deletion gate used to track whether the remaining Python test surface can be deleted yet
- the checked-in `docs/reports/pytest-harness-status.json` snapshot when `--report-path` is supplied

The checked-in snapshot is not documentation-only: `go test ./internal/regression` now re-computes the live Go-owned report and fails if `docs/reports/pytest-harness-status.json` drifts from the current repository state.

The snapshot intentionally stores repo-relative path fields (`project_root: "."`, `conftest_path: "tests/conftest.py"`) so it remains stable across clones and workspace directories.

The snapshot also records whether the top-level `tests/conftest.py` still exists plus whether it imports `pytest`, defines fixtures/hooks, or declares `pytest_plugins`, so the delete gate cannot regress silently if that file is reintroduced.

## Regression commands

Primary validation for this issue:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923 && PYTHONPATH=src python3 -c "from bigclaw.mapping import map_priority; from bigclaw.models import Priority; assert map_priority('P0') == Priority.P0"
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go run ./cmd/bigclawctl legacy-python pytest --repo .. --python python3 -- -- tests/test_planning.py tests/test_audit_events.py -q
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/testharness ./internal/regression ./cmd/bigclawctl
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go run ./cmd/bigclawctl pytest-harness --project-root .. --report-path docs/reports/pytest-harness-status.json --json
```

Observed results for this issue:

- `PYTHONPATH=src python3 -c "from bigclaw.mapping import map_priority; from bigclaw.models import Priority; assert map_priority('P0') == Priority.P0"` passed on the latest issue branch state, confirming the remaining legacy `src/bigclaw` import surface still works without relying on a checked-in pytest module.
- `go test ./internal/testharness ./internal/regression ./cmd/bigclawctl` passed on the latest issue branch state, covering the Go-owned script-runtime replacement for `tests/test_validation_bundle_continuation_policy_gate.py` together with the harness/report regression gates and the CLI exposure for the remaining legacy pytest asset blockers.
- `go run ./cmd/bigclawctl pytest-harness --project-root .. --report-path docs/reports/pytest-harness-status.json --json` passed on the latest issue branch state, regenerated the checked-in snapshot, and confirmed `inventory_summary=tests=18 bigclaw_imports=18 pytest_imports=0 pytest_command_refs=0`, `pyproject_declares_pytest=false`, `pyproject_has_pytest_config=false`, `conftest_exists=false`, `conftest_delete_status.can_delete=true`, and `legacy_pytest_delete_status.can_delete=false`.

Deletion-readiness validation for the legacy Python harness, once migration is further along:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923 && python3 -m pytest tests
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./...
```
