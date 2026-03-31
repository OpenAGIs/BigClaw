## BIG-GO-1023

### Plan
- Reduce `src/bigclaw` residual Python file count in a scoped tranche by removing low-coupling modules first.
- Preserve the legacy Python import contract for `bigclaw.audit_events`, `bigclaw.dsl`, and `bigclaw.deprecation` through package-level compatibility exports.
- Keep behavior aligned with existing `bigclaw-go` implementations where those already exist.
- Continue the tranche by folding `bigclaw.utility_surfaces` into package-level compatibility exports and deleting the standalone module.
- Continue the tranche by folding `bigclaw.repo_surfaces`, `bigclaw.workspace_bootstrap`, `bigclaw.control_surfaces`, `bigclaw.evaluation`, `bigclaw.planning`, and `bigclaw.execution_contract` into package-level compatibility exports and deleting the standalone modules.
- Continue the tranche by folding `bigclaw.ui_review` into package-level compatibility exports and deleting the standalone module.
- Continue the tranche by folding `bigclaw.models` into package-level compatibility exports and deleting the standalone module.
- Continue the tranche by folding `bigclaw.observability` into package-level compatibility exports and deleting the standalone module.

### Acceptance
- Changes stay scoped to remaining `src/bigclaw` Python assets for this tranche.
- `.py` file count under `src/bigclaw` decreases.
- Legacy Python imports and existing tests for audit specs, workflow definition parsing, and deprecation warnings still pass.
- Legacy Python imports and existing tests for cost control, memory, roadmap, validation policy, parallel refill, and legacy shim still pass after `utility_surfaces` deletion.
- Legacy Python imports and existing tests for repo surfaces and workspace bootstrap still pass after standalone module deletion.
- Legacy Python imports and existing tests for governance, issue archive, risk, planning, and scheduler still pass after `control_surfaces` deletion.
- Legacy Python imports and existing tests for benchmark/replay evaluation and operations analytics still pass after `evaluation.py` deletion.
- Legacy Python imports and existing tests for planning, candidate gates, four-week execution plans, and repo rollout still pass after `planning.py` deletion.
- Legacy Python imports and existing tests for execution contract, repo governance, and operations API contract still pass after `execution_contract.py` deletion.
- Legacy Python imports and existing tests for UI review pack generation and bundle export still pass after `ui_review.py` deletion.
- Legacy Python imports and existing tests for task/risk/triage/workflow/billing model round trips still pass after `models.py` deletion.
- Legacy Python imports and existing tests for observability ledger, repo sync audit, and downstream report/runtime consumers still pass after `observability.py` deletion.
- Report the impact on Python/Go file counts and note any `pyproject`/`setup` impact.

### Validation
- `pytest tests/test_audit_events.py tests/test_dsl.py`
- `python -m pytest tests/test_legacy_shim.py`
- `python3 -m pytest tests/test_cost_control.py tests/test_memory.py tests/test_parallel_refill.py tests/test_roadmap.py tests/test_validation_policy.py tests/test_legacy_shim.py`
- `python3 -m pytest tests/test_repo_board.py tests/test_repo_collaboration.py tests/test_repo_gateway.py tests/test_repo_governance.py tests/test_repo_links.py tests/test_repo_registry.py tests/test_repo_triage.py tests/test_observability.py`
- `python3 -m pytest tests/test_workspace_bootstrap.py`
- `python3 -m pytest tests/test_governance.py tests/test_risk.py tests/test_planning.py tests/test_scheduler.py tests/test_audit_events.py`
- `python3 -m pytest tests/test_evaluation.py tests/test_operations.py`
- `python3 -m pytest tests/test_planning.py tests/test_repo_rollout.py`
- `python3 -m pytest tests/test_execution_contract.py tests/test_repo_governance.py`
- `python3 -m pytest tests/test_ui_review.py`
- `python3 -m pytest tests/test_models.py`
- `PYTHONPATH=src python3 - <<'PY' ... import smoke for bigclaw.models ... PY`
- `python3 -m pytest tests/test_observability.py tests/test_reports.py`
- `PYTHONPATH=src python3 - <<'PY' ... import smoke for bigclaw.observability ... PY`
- `cd bigclaw-go && go test ./internal/bootstrap ./internal/repo ./internal/regression`
- `cd bigclaw-go && go test ./internal/observability ./internal/workflow ./internal/regression`
- `find src/bigclaw -maxdepth 1 -name '*.py' | wc -l`
- `find bigclaw-go -type f -name '*.go' | wc -l`
