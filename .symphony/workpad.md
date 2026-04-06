# BIG-GO-1542

## Plan
1. Capture the baseline repository-wide `.py` count and the exact tracked list for the remaining `tests/**/*.py` sweep surface.
2. Physically delete only the remaining Python test files under `tests/`.
3. Recount after deletion and record the exact removed-file list.
4. Run targeted validation commands for the file inventory and git deletion set.
5. Commit the scoped change set and push `BIG-GO-1542` to `origin`.

## Acceptance
- Remaining Python test files under `tests/` are physically removed.
- Before and after `.py` counts are captured.
- The exact removed-file list is captured.
- Validation commands and results are recorded.
- Changes stay scoped to this issue and are pushed to `origin/BIG-GO-1542`.

## Validation
- `find . -path './.git' -prune -o -type f -name '*.py' | sed 's#^./##' | sort`
- `git ls-files '*.py' | sort`
- `git ls-files 'tests/*.py' 'tests/**/*.py' | sort`
- `git diff --name-only --diff-filter=D | sort`
- `git status --short`

## Results
- Repository `.py` file count before deletion: `138`
- Repository `.py` file count after deletion: `81`
- In-scope `tests/**/*.py` count before deletion: `57`
- In-scope `tests/**/*.py` count after deletion: `0`
- Exact removed-file count: `57`

## Command Log
- `find . -path './.git' -prune -o -type f -name '*.py' -print | sed 's#^./##' | sort | wc -l | tr -d ' '` -> `138` before in `/Users/openagi/code/bigclaw-workspaces/BIG-GO-1548`, `81` after in `/Users/openagi/code/bigclaw-workspaces/BIG-GO-1542.repo`
- `git ls-files 'tests/*.py' 'tests/**/*.py' | sort | wc -l | tr -d ' '` -> `57`
- `find tests -type f -name '*.py' | sort | wc -l | tr -d ' '` -> `0`
- `git diff --name-only --diff-filter=D | sort | wc -l | tr -d ' '` -> `57`
- `cd bigclaw-go && go test ./...` -> `PASS` with all listed packages succeeding and `bigclaw-go/scripts/e2e` reporting `[no test files]`

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
