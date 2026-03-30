# BIG-GO-1005 Validation

## Batch Inventory

Baseline `src/bigclaw/**/*.py` files in scope:

- `src/bigclaw/__init__.py`
- `src/bigclaw/__main__.py`
- `src/bigclaw/audit_events.py`
- `src/bigclaw/collaboration.py`
- `src/bigclaw/connectors.py`
- `src/bigclaw/console_ia.py`
- `src/bigclaw/cost_control.py`
- `src/bigclaw/dashboard_run_contract.py`
- `src/bigclaw/deprecation.py`
- `src/bigclaw/design_system.py`
- `src/bigclaw/dsl.py`
- `src/bigclaw/evaluation.py`
- `src/bigclaw/event_bus.py`
- `src/bigclaw/execution_contract.py`
- `src/bigclaw/github_sync.py`
- `src/bigclaw/governance.py`
- `src/bigclaw/issue_archive.py`
- `src/bigclaw/legacy_shim.py`
- `src/bigclaw/mapping.py`
- `src/bigclaw/memory.py`
- `src/bigclaw/models.py`
- `src/bigclaw/observability.py`
- `src/bigclaw/operations.py`
- `src/bigclaw/parallel_refill.py`
- `src/bigclaw/pilot.py`
- `src/bigclaw/planning.py`
- `src/bigclaw/repo_board.py`
- `src/bigclaw/repo_commits.py`
- `src/bigclaw/repo_gateway.py`
- `src/bigclaw/repo_governance.py`
- `src/bigclaw/repo_links.py`
- `src/bigclaw/repo_plane.py`
- `src/bigclaw/repo_registry.py`
- `src/bigclaw/repo_triage.py`
- `src/bigclaw/reports.py`
- `src/bigclaw/risk.py`
- `src/bigclaw/roadmap.py`
- `src/bigclaw/run_detail.py`
- `src/bigclaw/runtime.py`
- `src/bigclaw/saved_views.py`
- `src/bigclaw/ui_review.py`
- `src/bigclaw/validation_policy.py`
- `src/bigclaw/workspace_bootstrap.py`
- `src/bigclaw/workspace_bootstrap_cli.py`
- `src/bigclaw/workspace_bootstrap_validation.py`

Remaining `src/bigclaw/**/*.py` files after this batch:

- `src/bigclaw/__init__.py`
- `src/bigclaw/__main__.py`
- `src/bigclaw/audit_events.py`
- `src/bigclaw/collaboration.py`
- `src/bigclaw/connectors.py`
- `src/bigclaw/console_ia.py`
- `src/bigclaw/dashboard_run_contract.py`
- `src/bigclaw/deprecation.py`
- `src/bigclaw/design_system.py`
- `src/bigclaw/dsl.py`
- `src/bigclaw/evaluation.py`
- `src/bigclaw/event_bus.py`
- `src/bigclaw/execution_contract.py`
- `src/bigclaw/github_sync.py`
- `src/bigclaw/governance.py`
- `src/bigclaw/issue_archive.py`
- `src/bigclaw/legacy_shim.py`
- `src/bigclaw/memory.py`
- `src/bigclaw/models.py`
- `src/bigclaw/observability.py`
- `src/bigclaw/operations.py`
- `src/bigclaw/planning.py`
- `src/bigclaw/repo_board.py`
- `src/bigclaw/repo_commits.py`
- `src/bigclaw/repo_gateway.py`
- `src/bigclaw/repo_links.py`
- `src/bigclaw/repo_plane.py`
- `src/bigclaw/repo_registry.py`
- `src/bigclaw/reports.py`
- `src/bigclaw/risk.py`
- `src/bigclaw/run_detail.py`
- `src/bigclaw/runtime.py`
- `src/bigclaw/saved_views.py`
- `src/bigclaw/ui_review.py`
- `src/bigclaw/workspace_bootstrap.py`
- `src/bigclaw/workspace_bootstrap_validation.py`

## Delete / Replace / Keep

Deleted in this batch:

- `src/bigclaw/cost_control.py`
  - deleted because no in-repo tests, scripts, or package exports still depended on it
  - replaced by `bigclaw-go/internal/costcontrol/controller.go`
- `src/bigclaw/parallel_refill.py`
  - deleted because refill queue operations already run through `bigclawctl refill`
  - replaced by `bigclaw-go/internal/refill/queue.go`, `bigclaw-go/internal/refill/queue_markdown.go`, and `bigclaw-go/cmd/bigclawctl/main.go`
- `src/bigclaw/pilot.py`
  - deleted because no Python runtime path or package export still referenced it
  - replaced by `bigclaw-go/internal/pilot/report.go`
- `src/bigclaw/workspace_bootstrap_cli.py`
  - deleted because workspace bootstrap/cleanup/validate already run through the Go CLI
  - replaced by `bigclaw-go/cmd/bigclawctl/main.go` and `bigclaw-go/internal/bootstrap/bootstrap.go`
- `src/bigclaw/validation_policy.py`
  - deleted because its compatibility surface is now registered from `src/bigclaw/__init__.py`
  - replaced by `bigclaw.__init__` compatibility wiring for `bigclaw.validation_policy`
- `src/bigclaw/repo_triage.py`
  - deleted because its compatibility surface is now registered from `src/bigclaw/__init__.py`
  - replaced by `bigclaw.__init__` compatibility wiring for `bigclaw.repo_triage` and `bigclaw-go/internal/repo/triage.go`
- `src/bigclaw/repo_governance.py`
  - deleted because its compatibility surface is now registered from `src/bigclaw/__init__.py`
  - replaced by `bigclaw.__init__` compatibility wiring for `bigclaw.repo_governance` and `bigclaw-go/internal/repo/governance.go`
- `src/bigclaw/mapping.py`
  - deleted because its compatibility surface is now registered from `src/bigclaw/__init__.py`
  - replaced by `bigclaw.__init__` compatibility wiring for `bigclaw.mapping` and `bigclaw-go/internal/intake/mapping.go`
- `src/bigclaw/roadmap.py`
  - deleted because its compatibility surface is now registered from `src/bigclaw/__init__.py`
  - replaced by `bigclaw.__init__` compatibility wiring for `bigclaw.roadmap`

Kept in this batch:

- `src/bigclaw/workspace_bootstrap_validation.py`
  - kept because [tests/test_workspace_bootstrap.py](/Users/openagi/code/bigclaw-workspaces/BIG-GO-1005/tests/test_workspace_bootstrap.py) imports it directly
- `src/bigclaw/repo_gateway.py`
  - kept because [tests/test_repo_gateway.py](/Users/openagi/code/bigclaw-workspaces/BIG-GO-1005/tests/test_repo_gateway.py) still validates the Python surface directly
- `src/bigclaw/github_sync.py`
  - kept because [tests/test_github_sync.py](/Users/openagi/code/bigclaw-workspaces/BIG-GO-1005/tests/test_github_sync.py) still exercises the Python compatibility surface
- `src/bigclaw/issue_archive.py`
  - kept because `bigclaw.__init__` still re-exports the archive models and report renderer from a dedicated module
- all other listed `src/bigclaw/*.py`
  - kept because this batch stayed scoped to modules with no remaining in-repo Python coupling or larger active compatibility surfaces

## Python File Count Impact

- Repo-wide Python files before: `108`
- Repo-wide Python files after: `99`
- Net reduction: `9`

- `src/bigclaw/*.py` files before: `45`
- `src/bigclaw/*.py` files after: `36`
- Net reduction inside batch: `9`

## Validation Commands

- `rg --files src/bigclaw -g '*.py'`
- `find . -name '*.py' | wc -l`
- `python3 -m compileall src/bigclaw`
- `python3 -m pytest tests/test_workspace_bootstrap.py tests/test_github_sync.py tests/test_validation_policy.py tests/test_repo_triage.py`
- `python3 -m pytest tests/test_validation_policy.py tests/test_repo_triage.py tests/test_repo_governance.py tests/test_repo_rollout.py`
- `python3 - <<'PY' ... PY`

## Validation Results

- `rg --files src/bigclaw -g '*.py'`
  - pass; confirmed the deleted files no longer exist and `src/bigclaw` now contains `36` Python files
- `find . -name '*.py' | wc -l`
  - pass; repo-wide Python file count dropped from `108` to `99`
- `python3 -m compileall src/bigclaw`
  - pass
- `python3 -m pytest tests/test_workspace_bootstrap.py tests/test_github_sync.py tests/test_validation_policy.py tests/test_repo_triage.py`
  - pass; `18 passed in 4.02s`
- `python3 -m pytest tests/test_validation_policy.py tests/test_repo_triage.py tests/test_repo_governance.py tests/test_repo_rollout.py`
  - pass; `8 passed in 0.09s`
- `python3 - <<'PY' ... PY`
  - pass; confirmed direct imports still work for `bigclaw.mapping`, `bigclaw.roadmap`, and `bigclaw.validation_policy` after moving those surfaces into `bigclaw.__init__`
