## BIG-GO-1023

### Plan
- Reduce `src/bigclaw` residual Python file count in a scoped tranche by removing low-coupling modules first.
- Preserve the legacy Python import contract for `bigclaw.audit_events`, `bigclaw.dsl`, and `bigclaw.deprecation` through package-level compatibility exports.
- Keep behavior aligned with existing `bigclaw-go` implementations where those already exist.
- Continue the tranche by folding `bigclaw.utility_surfaces` into package-level compatibility exports and deleting the standalone module.
- Continue the tranche by folding `bigclaw.repo_surfaces`, `bigclaw.workspace_bootstrap`, and `bigclaw.control_surfaces` into package-level compatibility exports and deleting the standalone modules.

### Acceptance
- Changes stay scoped to remaining `src/bigclaw` Python assets for this tranche.
- `.py` file count under `src/bigclaw` decreases.
- Legacy Python imports and existing tests for audit specs, workflow definition parsing, and deprecation warnings still pass.
- Legacy Python imports and existing tests for cost control, memory, roadmap, validation policy, parallel refill, and legacy shim still pass after `utility_surfaces` deletion.
- Legacy Python imports and existing tests for repo surfaces and workspace bootstrap still pass after standalone module deletion.
- Legacy Python imports and existing tests for governance, issue archive, risk, planning, and scheduler still pass after `control_surfaces` deletion.
- Report the impact on Python/Go file counts and note any `pyproject`/`setup` impact.

### Validation
- `pytest tests/test_audit_events.py tests/test_dsl.py`
- `python -m pytest tests/test_legacy_shim.py`
- `python3 -m pytest tests/test_cost_control.py tests/test_memory.py tests/test_parallel_refill.py tests/test_roadmap.py tests/test_validation_policy.py tests/test_legacy_shim.py`
- `python3 -m pytest tests/test_repo_board.py tests/test_repo_collaboration.py tests/test_repo_gateway.py tests/test_repo_governance.py tests/test_repo_links.py tests/test_repo_registry.py tests/test_repo_triage.py tests/test_observability.py`
- `python3 -m pytest tests/test_workspace_bootstrap.py`
- `python3 -m pytest tests/test_governance.py tests/test_risk.py tests/test_planning.py tests/test_scheduler.py tests/test_audit_events.py`
- `cd bigclaw-go && go test ./internal/bootstrap ./internal/repo ./internal/regression`
- `cd bigclaw-go && go test ./internal/observability ./internal/workflow ./internal/regression`
- `find src/bigclaw -maxdepth 1 -name '*.py' | wc -l`
- `find bigclaw-go -type f -name '*.go' | wc -l`
