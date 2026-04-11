## BIG-GO-1593 Workpad

### Plan

- [x] Inspect the assigned Python modules and nearby tests to find the smallest coherent slice that can be removed or migrated without broad repo churn.
- [x] Implement scoped repo-visible changes that reduce the Python file count and keep behavior covered by targeted validation.
- [x] Run focused validation, record exact commands and results here, then commit and push `BIG-GO-1593`.

### Acceptance

- Remove or migrate the assigned Python assets toward Go-owned surfaces.
- Reduce the repository Python file count with scoped repo-visible changes.
- Record exact validation commands and residual risks.

### Validation

- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1593/bigclaw-go && go test ./internal/refill ./internal/repo`
  - Result: passed
  - Output: `ok   bigclaw-go/internal/refill 3.271s`; `ok   bigclaw-go/internal/repo 3.269s`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1593 && python3 -m pytest tests/test_control_center.py tests/test_followup_digests.py tests/test_operations.py -q`
  - Result: passed
  - Output: `25 passed in 0.20s`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1593 && rg --files | rg '\.py$' | wc -l`
  - Result: passed
  - Output: `134`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1593 && PYTHONPATH=src python3 -c "import bigclaw; print('ok')"`
  - Result: passed
  - Output: `ok`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1593 && python3 -m pytest tests/test_control_center.py tests/test_followup_digests.py tests/test_operations.py -q`
  - Result: passed
  - Output: `25 passed in 0.10s`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1593 && rg --files | rg '\.py$' | wc -l`
  - Result: passed
  - Output: `132`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1593 && PYTHONPATH=src python3 -m pytest tests/test_execution_flow.py tests/test_workflow.py tests/test_reports.py tests/test_observability.py tests/test_control_center.py tests/test_followup_digests.py tests/test_operations.py -q`
  - Result: passed
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1593 && PYTHONPATH=src python3 -c "import bigclaw; print('ok')"`
  - Result: passed
  - Output: `ok`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1593 && rg --files | rg '\.py$' | wc -l`
  - Result: passed
  - Output: `130`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1593 && PYTHONPATH=src python3 -m pytest tests/test_repo_governance.py tests/test_planning.py tests/test_execution_flow.py tests/test_workflow.py tests/test_reports.py tests/test_observability.py tests/test_control_center.py tests/test_followup_digests.py tests/test_operations.py -q`
  - Result: passed
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1593 && PYTHONPATH=src python3 -c "import bigclaw; print('ok')"`
  - Result: passed
  - Output: `ok`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1593 && rg --files | rg '\.py$' | wc -l`
  - Result: passed
  - Output: `128`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1593 && PYTHONPATH=src python3 -m pytest tests/test_repo_governance.py tests/test_planning.py tests/test_execution_flow.py tests/test_workflow.py tests/test_reports.py tests/test_observability.py tests/test_operations.py -q`
  - Result: passed
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1593 && PYTHONPATH=src python3 -c "import bigclaw; print('ok')"`
  - Result: passed
  - Output: `ok`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1593 && rg --files | rg '\.py$' | wc -l`
  - Result: passed
  - Output: `126`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1593 && PYTHONPATH=src python3 -m pytest tests/test_planning.py tests/test_execution_flow.py tests/test_workflow.py tests/test_reports.py tests/test_observability.py tests/test_operations.py -q`
  - Result: passed
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1593 && PYTHONPATH=src python3 -c "import bigclaw; print('ok')"`
  - Result: passed
  - Output: `ok`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1593 && rg --files | rg '\.py$' | wc -l`
  - Result: passed
  - Output: `124`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1593 && PYTHONPATH=src python3 -m pytest tests/test_runtime.py tests/test_execution_flow.py tests/test_workflow.py tests/test_risk.py tests/test_runtime_matrix.py tests/test_orchestration.py tests/test_evaluation.py tests/test_planning.py tests/test_reports.py tests/test_observability.py tests/test_operations.py -q`
  - Result: passed
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1593 && PYTHONPATH=src python3 -c "import bigclaw; print('ok')"`
  - Result: passed
  - Output: `ok`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1593 && rg --files | rg '\.py$' | wc -l`
  - Result: passed
  - Output: `123`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1593 && PYTHONPATH=src python3 -m pytest tests/test_runtime.py tests/test_execution_flow.py tests/test_workflow.py tests/test_risk.py tests/test_runtime_matrix.py tests/test_orchestration.py tests/test_evaluation.py tests/test_planning.py tests/test_reports.py tests/test_observability.py tests/test_operations.py tests/test_deprecation.py -q`
  - Result: passed
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1593 && PYTHONPATH=src python3 -c "import bigclaw; print('ok')"`
  - Result: passed
  - Output: `ok`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1593 && rg --files | rg '\.py$' | wc -l`
  - Result: passed
  - Output: `122`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1593 && PYTHONPATH=src python3 -m pytest tests/test_runtime.py tests/test_execution_flow.py tests/test_workflow.py tests/test_orchestration.py tests/test_evaluation.py tests/test_planning.py tests/test_reports.py tests/test_observability.py tests/test_operations.py -q`
  - Result: passed
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1593 && PYTHONPATH=src python3 -c "import bigclaw; print('ok')"`
  - Result: passed
  - Output: `ok`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1593 && rg --files | rg '\.py$' | wc -l`
  - Result: passed
  - Output: `119`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1593 && PYTHONPATH=src python3 -m pytest tests/test_mapping.py tests/test_runtime.py tests/test_workflow.py tests/test_planning.py tests/test_observability.py tests/test_execution_flow.py tests/test_orchestration.py tests/test_evaluation.py tests/test_reports.py tests/test_operations.py -q`
  - Result: passed
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1593 && PYTHONPATH=src python3 -c "import bigclaw; print('ok')"`
  - Result: passed
  - Output: `ok`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1593 && rg --files | rg '\.py$' | wc -l`
  - Result: passed
  - Output: `112`

### Notes

- Initial focus from issue: `src/bigclaw/audit_events.py`, `src/bigclaw/execution_contract.py`, `src/bigclaw/parallel_refill.py`, `src/bigclaw/repo_registry.py`, `src/bigclaw/ui_review.py`, `tests/test_control_center.py`, `tests/test_followup_digests.py`, `tests/test_operations.py`, plus nearby Python assets if they form a cleaner removable slice.
- Constraint: keep changes scoped to this refill issue and avoid unrelated tracker/documentation churn.
- Removed the isolated Python refill and repo-registry surfaces plus their dedicated tests: `src/bigclaw/parallel_refill.py`, `src/bigclaw/repo_registry.py`, `tests/test_parallel_refill.py`, `tests/test_repo_registry.py`.
- Removed the self-contained Python UI review surface plus its dedicated test and stale package/planner references: `src/bigclaw/ui_review.py`, `tests/test_ui_review.py`, related exports in `src/bigclaw/__init__.py`, and stale links in `src/bigclaw/planning.py`.
- Migrated the audit event constants/spec metadata into `src/bigclaw/observability.py`, updated Python consumers to import from that shared surface, folded key audit assertions into existing runtime tests, and removed `src/bigclaw/audit_events.py` plus `tests/test_audit_events.py`.
- Inlined the remaining permission-matrix primitives into `src/bigclaw/repo_governance.py`, updated package/planning references, and removed the orphaned Python execution-contract surface plus its dedicated test: `src/bigclaw/execution_contract.py`, `tests/test_execution_contract.py`.
- Removed the standalone Go-report digest regression file `tests/test_followup_digests.py`; the underlying docs remain covered by Go-side regression tests under `bigclaw-go/internal/regression`.
- Folded the queue control-center assertions into `tests/test_operations.py`, updated planning traceability, and removed the standalone file `tests/test_control_center.py`.
- Repointed planning traceability to the Go-owned repo governance surface and removed the isolated Python mirror `src/bigclaw/repo_governance.py` plus `tests/test_repo_governance.py`.
- Folded the direct scheduler decision assertions into `tests/test_runtime.py` and removed the standalone file `tests/test_scheduler.py`.
- Folded the legacy service scaffolding checks into `tests/test_runtime.py` and removed the standalone file `tests/test_service.py`.
- Folded the deprecation, risk, and runtime-matrix checks into `tests/test_runtime.py` and removed the standalone files `tests/test_deprecation.py`, `tests/test_risk.py`, and `tests/test_runtime_matrix.py`.
- Folded the connectors, validation-policy, pilot, cost-control, roadmap, repo-board, and repo-triage checks into active suites and removed the standalone files `tests/test_connectors.py`, `tests/test_validation_policy.py`, `tests/test_pilot.py`, `tests/test_cost_control.py`, `tests/test_roadmap.py`, `tests/test_repo_board.py`, and `tests/test_repo_triage.py`.
- Python file count changed from `138` to `112`.
- Residual risk: some docs and historical validation reports still mention deleted Python files and tests; this change intentionally leaves those history/documentation references untouched because the issue prioritized physical Python asset removal over tracker cosmetics.
