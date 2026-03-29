# BIG-GO-964 Workpad

## Plan

1. Inventory the remaining `src/bigclaw/*.py` application modules, group the smallest modules by dependency direction, and pick a consolidation slice that reduces file count without changing runtime behavior.
2. Move the selected small-module implementations into existing neighboring modules, then preserve the legacy `bigclaw.<module>` import surface through package-level module aliases.
3. Delete the superseded Python files, update direct internal imports and package re-exports, and add focused regression coverage for the legacy import paths plus the migrated behavior.
4. Run targeted validation commands for the touched modules, record exact commands and results, then verify the resulting `src/bigclaw/*.py` count delta.
5. Commit the scoped change set for `BIG-GO-964` and push the branch to the remote.

## Acceptance

- Produce an explicit list of the Python files directly handled in this issue.
- Reduce the number of Python files under `src/bigclaw/` for this batch.
- Document delete / replace / retain reasoning for the handled files.
- Report the before/after impact on the overall `src/bigclaw/*.py` file count.

## Validation

- `pytest` for the exact test modules covering the consolidated Python surfaces.
- A direct Python import check for legacy `bigclaw.<module>` names that are now served by aliases.
- `git status --short` to confirm the change set stays scoped to this issue before commit.

## Results

- Directly handled deleted Python files in `src/bigclaw/`:
  - `deprecation.py`
  - `dsl.py`
  - `event_bus.py`
  - `github_sync.py`
  - `legacy_shim.py`
  - `connectors.py`
  - `mapping.py`
  - `memory.py`
  - `parallel_refill.py`
  - `cost_control.py`
  - `pilot.py`
  - `repo_board.py`
  - `repo_commits.py`
  - `repo_gateway.py`
  - `repo_governance.py`
  - `repo_links.py`
  - `repo_registry.py`
  - `repo_triage.py`
  - `roadmap.py`
  - `risk.py`
  - `runtime.py`
  - `service.py`
  - `validation_policy.py`
  - `workspace_bootstrap_cli.py`
  - `workspace_bootstrap_validation.py`
  - `audit_events.py`
  - `orchestration.py`
- Replacement / consolidation targets:
  - `legacy_shim.py` now owns the legacy runtime deprecation helpers.
  - `collaboration.py` now owns the repo discussion board helpers.
  - `models.py` now owns the connector stubs and source-issue mapping helpers.
  - `operations.py` now owns the budget control helpers.
  - `observability.py` now owns the canonical audit event and event bus helpers.
  - `queue.py` now owns the parallel refill queue helpers.
  - `queue.py` now owns the task memory helpers.
  - `planning.py` now owns the execution-pack roadmap dataclasses and builder.
  - `repo_plane.py` now owns the repo commit, gateway, governance, link, registry, and triage surfaces.
  - `reports.py` now owns the pilot implementation and validation report policy helpers.
  - `reports.py` now owns the orchestration planning and policy helpers.
  - `scheduler.py` now owns the risk scoring helpers.
  - `scheduler.py` now owns the worker runtime helpers.
  - `workflow.py` now owns the workflow DSL helpers.
  - `workspace_bootstrap.py` now owns the git sync automation helpers.
  - `workspace_bootstrap.py` now owns the legacy shim helpers.
  - `workspace_bootstrap.py` now owns the bootstrap validation helpers.
  - `workspace_bootstrap.py` now owns the bootstrap CLI wrapper helpers.
  - `__main__.py` now owns the legacy HTTP service helpers.
  - `__init__.py` now registers compatibility aliases so `import bigclaw.<old_module>` still resolves.
- Retained nearby Python files and reasons:
  - `execution_contract.py`: retained as the generic permission-contract host; repo policy compatibility now aliases into `repo_plane.py` without widening into broader contract semantics.
  - `reports.py`: retained as the primary reporting host after absorbing pilot and validation helpers; further consolidation there would stop being low-risk.
  - `scheduler.py`: retained as the execution host after absorbing risk and runtime helpers; orchestration moved into `reports.py` instead of broadening scheduler/report cyclic ownership.
  - `operations.py`: retained as the operations-policy host after absorbing budget control helpers; broader merging beyond this would widen the issue.
  - `observability.py`: retained as the runtime evidence host after absorbing audit and event bus helpers; broader collapsing here would stop being low-risk.
  - `workspace_bootstrap.py`: retained as the bootstrap/cache host after absorbing validation helpers; further collapsing this area would couple CLI/runtime surfaces more tightly.
  - `__main__.py`: retained as the package execution entrypoint; deleting it would remove `python -m bigclaw` compatibility instead of just compressing internals.
- Python file count impact under `src/bigclaw/*.py`:
  - Before: `49`
  - After: `23`
  - Delta: `-26`
- Exact validation commands and results:
  - `PYTHONPATH=src python3 - <<'PY' ... importlib.import_module(...) ... PY`
    - Result: legacy imports resolved successfully:
      `bigclaw.deprecation -> bigclaw.legacy_shim`
      `bigclaw.mapping -> bigclaw.connectors`
      `bigclaw.parallel_refill -> bigclaw.queue`
      `bigclaw.roadmap -> bigclaw.planning`
      `bigclaw.repo_commits -> bigclaw.repo_plane`
      `bigclaw.repo_gateway -> bigclaw.repo_plane`
      `bigclaw.repo_governance -> bigclaw.repo_plane`
      `bigclaw.repo_links -> bigclaw.repo_plane`
      `bigclaw.repo_registry -> bigclaw.repo_plane`
      `bigclaw.repo_triage -> bigclaw.repo_plane`
  - `PYTHONPATH=src python3 -m pytest tests/test_mapping.py tests/test_connectors.py tests/test_queue.py tests/test_repo_gateway.py tests/test_repo_governance.py tests/test_repo_links.py tests/test_repo_registry.py tests/test_repo_triage.py tests/test_planning.py`
    - Result: `30 passed in 0.12s`
  - `PYTHONPATH=src python3 -m pytest tests/test_validation_policy.py tests/test_operations.py tests/test_reports.py`
    - Result: `56 passed in 0.17s`
  - `PYTHONPATH=src python3 -m pytest tests/test_repo_board.py tests/test_repo_collaboration.py tests/test_workspace_bootstrap.py`
    - Result: `11 passed in 3.07s`
  - `PYTHONPATH=src python3 -m pytest tests/test_risk.py tests/test_dsl.py tests/test_audit_events.py tests/test_event_bus.py tests/test_memory.py`
    - Result: `16 passed in 0.10s`
  - `PYTHONPATH=src python3 -m pytest tests/test_scheduler.py tests/test_workflow.py tests/test_runtime.py tests/test_observability.py tests/test_queue.py`
    - Result: `28 passed in 0.11s`
  - `PYTHONPATH=src python3 -m pytest tests/test_workspace_bootstrap.py`
    - Result: `9 passed in 3.02s`
  - `PYTHONPATH=src python3 -m pytest tests/test_github_sync.py tests/test_workspace_bootstrap.py tests/test_runtime.py tests/test_workflow.py tests/test_scheduler.py tests/test_observability.py`
    - Result: `38 passed in 3.97s`
  - `PYTHONPATH=src python3 -m pytest tests/test_workspace_bootstrap.py tests/test_github_sync.py tests/test_runtime.py tests/test_workflow.py tests/test_scheduler.py tests/test_observability.py tests/test_queue.py`
    - Result: `42 passed in 4.07s`
  - `PYTHONPATH=src python3 -m pytest tests/test_connectors.py tests/test_mapping.py tests/test_models.py`
    - Result: `7 passed in 0.08s`
  - `PYTHONPATH=src python3 -m pytest tests/test_runtime.py tests/test_runtime_matrix.py tests/test_scheduler.py tests/test_workflow.py tests/test_observability.py`
    - Result: `27 passed in 0.10s`
  - `PYTHONPATH=src python3 - <<'PY' ... importlib.import_module(...) ... PY`
    - Result: legacy orchestration import resolved successfully:
      `bigclaw.orchestration -> bigclaw.reports`
      `bigclaw.reports -> bigclaw.reports`
      `bigclaw.scheduler -> bigclaw.scheduler`
      `bigclaw.workflow -> bigclaw.workflow`
  - `PYTHONPATH=src python3 -m pytest tests/test_orchestration.py tests/test_reports.py tests/test_scheduler.py tests/test_workflow.py tests/test_observability.py`
    - Result: `58 passed in 0.12s`
  - `python3 -m py_compile src/bigclaw/*.py`
    - Result: passed with no output
  - `find src/bigclaw -maxdepth 1 -name '*.py' | wc -l`
    - Result: `23`
