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

This means the legacy pytest harness is an import bootstrap, not a fixture/runtime orchestration layer.

## Go replacement landed in this issue

The new Go-native baseline lives in `bigclaw-go/internal/testharness`.

It provides:

- `RepoRoot(tb)` to locate the `bigclaw-go` module root without relying on package cwd
- `ProjectRoot(tb)` to reach the parent repository root that still contains legacy `src/` and `tests/`
- `JoinRepoRoot(tb, elems...)` and `JoinProjectRoot(tb, elems...)` for stable fixture/report path resolution
- `ResolveProjectPath(tb, candidate)` for paths that may still be prefixed with `bigclaw-go/`
- `PrependPathEnv(tb, dir)` for path-based CLI bootstrapping
- `Chdir(tb, dir)` for temporary cwd changes with automatic cleanup

First-batch adoption landed here:

- `internal/regression/*_test.go` now uses the shared repo-root baseline instead of ad hoc `../..` resolution and `runtime.Caller` plumbing
- `cmd/bigclawctl/migration_commands_test.go` now uses the shared cwd and `PATH` bootstrap helpers

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

Still partially migrated for broader model-runtime semantics:

- `tests/test_models.py` now overlaps with Go coverage in `bigclaw-go/internal/workflow/model_test.go` and `bigclaw-go/internal/billing/statement_test.go`
- The remaining Python-owned pieces are richer risk-assessment and triage record model surfaces that do not have a single equivalent Go model package in this issue

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

Until then, `tests/conftest.py` remains a compatibility shim and should not grow new behavior.

## Regression commands

Primary validation for this issue:

```bash
cd bigclaw-go && go test ./internal/testharness ./internal/regression ./cmd/bigclawctl
cd bigclaw-go && go test ./...
```

Deletion-readiness validation for the legacy Python harness, once migration is further along:

```bash
python3 -m pytest tests
cd bigclaw-go && go test ./...
```
