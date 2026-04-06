# BIG-GO-1552 Evidence

## Scope

- Delete the remaining repository-root `tests/*.py` files from disk.
- Record the exact repository-wide `.py` before/after count delta.
- Record the exact removed-file ledger for the deleted test files.

## Before/After Counts

- Repository `.py` files: `115 -> 58` (`-57`)
- In-scope `tests/*.py` files: `57 -> 0` (`-57`)

## Removed Files

- `tests/conftest.py`
- `tests/test_audit_events.py`
- `tests/test_connectors.py`
- `tests/test_console_ia.py`
- `tests/test_control_center.py`
- `tests/test_cost_control.py`
- `tests/test_cross_process_coordination_surface.py`
- `tests/test_dashboard_run_contract.py`
- `tests/test_deprecation.py`
- `tests/test_design_system.py`
- `tests/test_dsl.py`
- `tests/test_evaluation.py`
- `tests/test_event_bus.py`
- `tests/test_execution_contract.py`
- `tests/test_execution_flow.py`
- `tests/test_followup_digests.py`
- `tests/test_github_sync.py`
- `tests/test_governance.py`
- `tests/test_issue_archive.py`
- `tests/test_legacy_shim.py`
- `tests/test_live_shadow_bundle.py`
- `tests/test_live_shadow_scorecard.py`
- `tests/test_mapping.py`
- `tests/test_memory.py`
- `tests/test_models.py`
- `tests/test_observability.py`
- `tests/test_operations.py`
- `tests/test_orchestration.py`
- `tests/test_parallel_refill.py`
- `tests/test_parallel_validation_bundle.py`
- `tests/test_pilot.py`
- `tests/test_planning.py`
- `tests/test_queue.py`
- `tests/test_repo_board.py`
- `tests/test_repo_collaboration.py`
- `tests/test_repo_gateway.py`
- `tests/test_repo_governance.py`
- `tests/test_repo_links.py`
- `tests/test_repo_registry.py`
- `tests/test_repo_rollout.py`
- `tests/test_repo_triage.py`
- `tests/test_reports.py`
- `tests/test_risk.py`
- `tests/test_roadmap.py`
- `tests/test_runtime.py`
- `tests/test_runtime_matrix.py`
- `tests/test_saved_views.py`
- `tests/test_scheduler.py`
- `tests/test_service.py`
- `tests/test_shadow_matrix_corpus.py`
- `tests/test_subscriber_takeover_harness.py`
- `tests/test_ui_review.py`
- `tests/test_validation_bundle_continuation_policy_gate.py`
- `tests/test_validation_bundle_continuation_scorecard.py`
- `tests/test_validation_policy.py`
- `tests/test_workflow.py`
- `tests/test_workspace_bootstrap.py`

## Validation

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort | wc -l` -> `58`
- `find tests -type f -name '*.py' | sort | wc -l` -> `0` (`find: tests: No such file or directory`)
- `git diff --cached --name-only --diff-filter=D | sort | wc -l` -> `57`
- `go test -count=1 ./internal/regression -run 'TestBIGGO1552(RepositoryPythonCountDrop|TestsPythonFilesDeleted|LaneReportCapturesExactCounts)$'` -> `ok  	bigclaw-go/internal/regression	1.164s`
