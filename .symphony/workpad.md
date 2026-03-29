# BIG-GO-964 Workpad

## Plan

1. Inventory the remaining `src/bigclaw/*.py` application modules, group the smallest modules by dependency direction, and pick a consolidation slice that reduces file count without changing runtime behavior.
2. Move the selected small-module implementations into existing neighboring modules, then preserve the legacy `bigclaw.<module>` import surface through package-level module aliases.
3. Delete the superseded Python files, update direct internal imports and package re-exports, and add focused regression coverage for the legacy import paths plus the migrated behavior.
4. Run targeted validation commands for the touched modules, record exact commands and results, then verify the resulting `src/bigclaw/*.py` count delta.
5. Commit the scoped change set for `BIG-GO-964` and push the branch to the remote.
6. Current slice: fold `issue_archive.py` into `reports.py`, keep the legacy `bigclaw.issue_archive` import path via aliasing, and add focused regression coverage for the archive surface.
7. Next slice: fold `governance.py` into `planning.py`, keep the legacy `bigclaw.governance` import path via aliasing, and validate both governance and planning behavior.
8. Follow-on slice: fold `dashboard_run_contract.py` into `execution_contract.py`, keep the legacy `bigclaw.dashboard_run_contract` import path via aliasing, and validate the dashboard/run contract surface.
9. Current next slice: fold `queue.py` into `operations.py`, keep the legacy `bigclaw.queue` import path via aliasing, and validate queue plus operations control-center behavior.
10. Follow-up slice: fold `run_detail.py` into `reports.py`, keep the legacy `bigclaw.run_detail` import path via aliasing, and validate the run-detail, evaluation, and reporting surfaces.
11. Current continuation slice: fold `evaluation.py` into `operations.py`, keep the legacy `bigclaw.evaluation` import path via aliasing, and validate evaluation plus operations regression surfaces.
12. Current next slice: fold `console_ia.py` into `design_system.py`, keep the legacy `bigclaw.console_ia` import path via aliasing, and validate console IA plus design-system regression surfaces.
13. Current next slice: fold `saved_views.py` into `reports.py`, keep the legacy `bigclaw.saved_views` import path via aliasing, and validate saved-view plus reporting regression surfaces.
14. Current next slice: fold `repo_plane.py` into `execution_contract.py`, keep the legacy `bigclaw.repo_plane` import path via aliasing, retarget direct internal imports, and validate repo-plane plus observability regression surfaces.
15. Current next slice: fold `workspace_bootstrap.py` into `__main__.py`, keep the legacy `bigclaw.workspace_bootstrap` import path via aliasing, retarget direct internal imports, preserve the package entrypoint separately from the bootstrap CLI surface, and validate bootstrap plus entrypoint regression surfaces.
16. Current next slice: fold `workflow.py` into `scheduler.py`, keep the legacy `bigclaw.workflow` import path via aliasing, and validate workflow, DSL, scheduler, and audit-event regression surfaces.
17. Current next slice: fold `models.py` into `execution_contract.py`, keep the legacy `bigclaw.models` import path via aliasing, retarget direct internal imports, and validate model, contract, scheduler, report, and observability regression surfaces.
18. Current next slice: fold `planning.py` into `reports.py`, keep the legacy `bigclaw.planning` import path via aliasing, and validate planning, governance, rollout, and reporting regression surfaces.
19. Current next slice: fold `ui_review.py` into `design_system.py`, keep the legacy `bigclaw.ui_review` import path via aliasing, and validate UI review plus design-system regression surfaces.

## Acceptance

- Produce an explicit list of the Python files directly handled in this issue.
- Reduce the number of Python files under `src/bigclaw/` for this batch.
- Document delete / replace / retain reasoning for the handled files.
- Report the before/after impact on the overall `src/bigclaw/*.py` file count.

## Validation

- `pytest` for the exact test modules covering the consolidated Python surfaces.
- A direct Python import check for legacy `bigclaw.<module>` names that are now served by aliases.
- `git status --short` to confirm the change set stays scoped to this issue before commit.
- For the current slice, validate `tests/test_issue_archive.py` plus `tests/test_reports.py` and re-check the top-level Python file count.
- For the next slice, validate `tests/test_governance.py` plus `tests/test_planning.py` and re-check the top-level Python file count.
- For the follow-on slice, validate `tests/test_dashboard_run_contract.py` plus `tests/test_execution_contract.py` and re-check the top-level Python file count.
- For the current next slice, validate `tests/test_queue.py`, `tests/test_control_center.py`, `tests/test_execution_flow.py`, and `tests/test_operations.py`, then re-check the top-level Python file count.
- For the follow-up slice, validate `tests/test_evaluation.py`, `tests/test_reports.py`, and `tests/test_observability.py`, then re-check the top-level Python file count.
- For the current continuation slice, validate `tests/test_evaluation.py` plus `tests/test_operations.py`, then re-check the top-level Python file count.
- For the current next slice, validate `tests/test_console_ia.py` plus `tests/test_design_system.py`, then re-check the top-level Python file count.
- For the current next slice, validate `tests/test_saved_views.py` plus `tests/test_reports.py`, then re-check the top-level Python file count.
- For the current next slice, validate `tests/test_repo_gateway.py`, `tests/test_repo_governance.py`, `tests/test_repo_links.py`, `tests/test_repo_registry.py`, `tests/test_repo_triage.py`, and `tests/test_observability.py`, then re-check the top-level Python file count.
- For the current next slice, validate `tests/test_workspace_bootstrap.py`, `tests/test_github_sync.py`, `tests/test_scheduler.py`, `tests/test_workflow.py`, and `tests/test_operations.py`, then re-check the top-level Python file count.
- For the current next slice, validate `tests/test_dsl.py`, `tests/test_workflow.py`, `tests/test_scheduler.py`, and `tests/test_audit_events.py`, then re-check the top-level Python file count.
- For the current next slice, validate `tests/test_models.py`, `tests/test_execution_contract.py`, `tests/test_scheduler.py`, `tests/test_reports.py`, and `tests/test_observability.py`, then re-check the top-level Python file count.
- For the current next slice, validate `tests/test_planning.py`, `tests/test_governance.py`, `tests/test_repo_rollout.py`, and `tests/test_reports.py`, then re-check the top-level Python file count.
- For the current next slice, validate `tests/test_ui_review.py` plus `tests/test_design_system.py`, then re-check the top-level Python file count.

## Results

- Directly handled deleted Python files in `src/bigclaw/`:
  - `deprecation.py`
  - `dsl.py`
  - `event_bus.py`
  - `github_sync.py`
  - `legacy_shim.py`
  - `connectors.py`
  - `collaboration.py`
  - `console_ia.py`
  - `saved_views.py`
  - `repo_plane.py`
  - `workspace_bootstrap.py`
  - `workflow.py`
  - `models.py`
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
  - `dashboard_run_contract.py`
  - `evaluation.py`
  - `governance.py`
  - `issue_archive.py`
  - `orchestration.py`
  - `planning.py`
  - `queue.py`
  - `run_detail.py`
  - `ui_review.py`
- Replacement / consolidation targets:
  - `execution_contract.py` now owns the dashboard/run schema contract helpers.
  - `execution_contract.py` now owns the repo permission, registry, commit-lineage, and triage helpers.
  - `execution_contract.py` now owns the shared task, flow, billing, connector, and source-issue mapping models.
  - `__main__.py` now owns the workspace bootstrap, validation, github-sync, and legacy runtime shim helpers.
  - `legacy_shim.py` now owns the legacy runtime deprecation helpers.
  - `models.py` now owns the connector stubs and source-issue mapping helpers.
  - `operations.py` now owns the budget control helpers.
  - `operations.py` now owns the benchmark evaluation and replay helpers.
  - `operations.py` now owns the queue persistence and parallel issue queue helpers.
  - `observability.py` now owns the canonical audit event and event bus helpers.
  - `observability.py` now owns the collaboration and repo discussion board helpers.
  - `design_system.py` now owns the console information architecture and interaction-contract helpers.
  - `design_system.py` now owns the UI review pack, signoff, blocker, and escalation rendering helpers.
  - `operations.py` now owns the task memory compatibility surface.
  - `planning.py` now owns the execution-pack roadmap dataclasses and builder.
  - `planning.py` now owns the scope-freeze governance helpers.
  - `repo_plane.py` now owns the repo commit, gateway, governance, link, registry, and triage surfaces.
  - `reports.py` now owns the pilot implementation and validation report policy helpers.
  - `reports.py` now owns the issue-priority archive helpers.
  - `reports.py` now owns the orchestration planning and policy helpers.
  - `reports.py` now owns the planning, roadmap, scope-freeze governance, candidate gate, and rollout scorecard helpers.
  - `reports.py` now owns the run-detail rendering helpers.
  - `reports.py` now owns the saved-view catalog and alert-digest helpers.
  - `scheduler.py` now owns the risk scoring helpers.
  - `scheduler.py` now owns the worker runtime helpers.
  - `scheduler.py` now owns the workflow definition, journal, acceptance-gate, and workflow-engine helpers.
  - `workflow.py` now owns the workflow DSL helpers.
  - `workspace_bootstrap.py` now owns the git sync automation helpers.
  - `workspace_bootstrap.py` now owns the legacy shim helpers.
  - `workspace_bootstrap.py` now owns the bootstrap validation helpers.
  - `workspace_bootstrap.py` now owns the bootstrap CLI wrapper helpers.
  - `__main__.py` now owns the legacy HTTP service helpers.
  - `__init__.py` now registers compatibility aliases so `import bigclaw.<old_module>` still resolves.
- Retained nearby Python files and reasons:
  - `execution_contract.py`: retained as the generic permission-contract host after absorbing dashboard/run schema contracts; repo policy compatibility now aliases into `repo_plane.py` without widening into unrelated control-plane semantics.
  - `execution_contract.py`: retained as the generic permission-contract host after also absorbing repo-plane permission and commit metadata helpers; this removes an existing repo-plane dependency on execution contracts instead of widening into unrelated operations/reporting ownership.
  - `execution_contract.py`: retained as the generic contract and shared-entity host after also absorbing models; this removes another existing dependency edge into execution contracts and centralizes the compatibility datamodels without widening into UI surfaces.
  - `reports.py`: retained as the primary reporting host after absorbing pilot, validation, issue-archive, and orchestration helpers; further consolidation there would stop being low-risk.
  - `reports.py`: retained as the primary reporting host after also absorbing saved-view catalog helpers; this keeps shared-view/reporting semantics co-located without widening into unrelated execution modules.
  - `reports.py`: retained as the closest remaining narrative and governance-adjacent host after also absorbing planning; this removes a standalone policy module without introducing a new internal dependency cycle.
  - `scheduler.py`: retained as the execution host after absorbing risk and runtime helpers; orchestration moved into `reports.py` instead of broadening scheduler/report cyclic ownership.
  - `scheduler.py`: retained as the execution host after also absorbing workflow engine helpers; this removes an existing dependency edge from workflow into scheduler and keeps execution orchestration on one host without widening into planning or operations.
  - `operations.py`: retained as the operations-policy host after absorbing budget control, queue, and memory helpers; broader merging beyond this would widen the issue.
  - `observability.py`: retained as the runtime evidence host after absorbing audit and event bus helpers; broader collapsing here would stop being low-risk.
  - `design_system.py`: retained as the UI specification host after absorbing console IA and interaction-contract helpers; the merge removes an existing dependency edge from console IA into design system without widening into saved-view or UI review ownership.
  - `design_system.py`: retained as the consolidated UI contract host after also absorbing UI review; this keeps adjacent product-specification and review artifacts together without leaking them into reports or scheduler surfaces.
  - `workspace_bootstrap.py`: retained as the bootstrap/cache host after absorbing validation helpers; further collapsing this area would couple CLI/runtime surfaces more tightly.
  - `__main__.py`: retained as the package execution entrypoint; deleting it would remove `python -m bigclaw` compatibility instead of just compressing internals.
  - `__main__.py`: retained as the package execution entrypoint after also absorbing workspace bootstrap and github-sync helpers; this keeps the migration-only runtime shim and related bootstrap CLI surfaces co-located while preserving `python -m bigclaw`.
- Python file count impact under `src/bigclaw/*.py`:
  - Before: `49`
  - After: `8`
  - Delta: `-41`
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
  - `PYTHONPATH=src python3 - <<'PY' ... importlib.import_module(...) ... PY`
    - Result: legacy issue archive import resolved successfully:
      `bigclaw.issue_archive -> bigclaw.reports`
      `bigclaw.reports -> bigclaw.reports`
  - `PYTHONPATH=src python3 -m pytest tests/test_issue_archive.py tests/test_reports.py`
    - Result: `38 passed in 0.10s`
  - `python3 -m py_compile src/bigclaw/*.py`
    - Result: passed with no output
  - `find src/bigclaw -maxdepth 1 -name '*.py' | wc -l`
    - Result: `22`
  - `PYTHONPATH=src python3 - <<'PY' ... importlib.import_module(...) ... PY`
    - Result: legacy governance import resolved successfully:
      `bigclaw.governance -> bigclaw.planning`
      `bigclaw.planning -> bigclaw.planning`
  - `PYTHONPATH=src python3 -m pytest tests/test_governance.py tests/test_planning.py`
    - Result: `18 passed in 0.08s`
  - `python3 -m py_compile src/bigclaw/*.py`
    - Result: passed with no output
  - `find src/bigclaw -maxdepth 1 -name '*.py' | wc -l`
    - Result: `21`
  - `PYTHONPATH=src python3 - <<'PY' ... importlib.import_module(...) ... PY`
    - Result: legacy dashboard contract import resolved successfully:
      `bigclaw.dashboard_run_contract -> bigclaw.execution_contract`
      `bigclaw.execution_contract -> bigclaw.execution_contract`
  - `PYTHONPATH=src python3 -m pytest tests/test_dashboard_run_contract.py tests/test_execution_contract.py`
    - Result: `10 passed in 0.09s`
  - `python3 -m py_compile src/bigclaw/*.py`
    - Result: passed with no output
  - `find src/bigclaw -maxdepth 1 -name '*.py' | wc -l`
    - Result: `20`
  - `PYTHONPATH=src python3 - <<'PY' ... importlib.import_module(...) ... PY`
    - Result: legacy queue and memory imports resolved successfully:
      `bigclaw.queue -> bigclaw.operations`
      `bigclaw.memory -> bigclaw.operations`
      `bigclaw.operations -> bigclaw.operations`
  - `PYTHONPATH=src python3 -m pytest tests/test_queue.py tests/test_memory.py tests/test_control_center.py tests/test_execution_flow.py tests/test_operations.py`
    - Result: `30 passed in 0.10s`
  - `python3 -m py_compile src/bigclaw/*.py`
    - Result: passed with no output
  - `find src/bigclaw -maxdepth 1 -name '*.py' | wc -l`
    - Result: `19`
  - `PYTHONPATH=src python3 - <<'PY' ... importlib.import_module(...) ... PY`
    - Result: legacy run detail import resolved successfully:
      `bigclaw.run_detail -> bigclaw.reports`
      `bigclaw.reports -> bigclaw.reports`
      `bigclaw.evaluation -> bigclaw.evaluation`
  - `PYTHONPATH=src python3 -m pytest tests/test_evaluation.py tests/test_reports.py tests/test_observability.py`
    - Result: `48 passed in 0.11s`
  - `python3 -m py_compile src/bigclaw/*.py`
    - Result: passed with no output
  - `find src/bigclaw -maxdepth 1 -name '*.py' | wc -l`
    - Result: `18`
  - `PYTHONPATH=src python3 - <<'PY' ... importlib.import_module(...) ... PY`
    - Result: legacy evaluation import resolved successfully:
      `bigclaw.evaluation -> bigclaw.operations`
      `bigclaw.operations -> bigclaw.operations`
  - `PYTHONPATH=src python3 -m pytest tests/test_evaluation.py tests/test_operations.py`
    - Result: `27 passed in 0.09s`
  - `python3 -m py_compile src/bigclaw/*.py`
    - Result: passed with no output
  - `find src/bigclaw -maxdepth 1 -name '*.py' | wc -l`
    - Result: `17`
  - `PYTHONPATH=src python3 - <<'PY' ... importlib.import_module(...) ... PY`
    - Result: legacy collaboration imports resolved successfully:
      `bigclaw.collaboration -> bigclaw.observability`
      `bigclaw.repo_board -> bigclaw.observability`
      `bigclaw.observability -> bigclaw.observability`
      `bigclaw.reports -> bigclaw.reports`
  - `PYTHONPATH=src python3 -m pytest tests/test_repo_collaboration.py tests/test_repo_board.py tests/test_observability.py tests/test_reports.py`
    - Result: `43 passed in 0.10s`
  - `python3 -m py_compile src/bigclaw/*.py`
    - Result: passed with no output
  - `find src/bigclaw -maxdepth 1 -name '*.py' | wc -l`
    - Result: `16`
  - `PYTHONPATH=src python3 - <<'PY' ... importlib.import_module(...) ... PY`
    - Result: legacy console IA import resolved successfully:
      `bigclaw.console_ia -> bigclaw.design_system`
      `bigclaw.design_system -> bigclaw.design_system`
      `bigclaw -> bigclaw`
  - `PYTHONPATH=src python3 -m pytest tests/test_console_ia.py tests/test_design_system.py`
    - Result: `26 passed in 0.10s`
  - `python3 -m py_compile src/bigclaw/*.py`
    - Result: passed with no output
  - `find src/bigclaw -maxdepth 1 -name '*.py' | wc -l`
    - Result: `15`
  - `PYTHONPATH=src python3 - <<'PY' ... importlib.import_module(...) ... PY`
    - Result: legacy saved-views import resolved successfully:
      `bigclaw.saved_views -> bigclaw.reports`
      `bigclaw.reports -> bigclaw.reports`
      `bigclaw -> bigclaw`
  - `PYTHONPATH=src python3 -m pytest tests/test_saved_views.py tests/test_reports.py`
    - Result: `38 passed in 0.09s`
  - `python3 -m py_compile src/bigclaw/*.py`
    - Result: passed with no output
  - `find src/bigclaw -maxdepth 1 -name '*.py' | wc -l`
    - Result: `14`
  - `PYTHONPATH=src python3 - <<'PY' ... importlib.import_module(...) ... PY`
    - Result: legacy repo-plane imports resolved successfully:
      `bigclaw.repo_plane -> bigclaw.execution_contract`
      `bigclaw.execution_contract -> bigclaw.execution_contract`
      `bigclaw.repo_governance -> bigclaw.execution_contract`
      `bigclaw.repo_links -> bigclaw.execution_contract`
      `bigclaw.repo_registry -> bigclaw.execution_contract`
      `bigclaw.repo_triage -> bigclaw.execution_contract`
  - `PYTHONPATH=src python3 -m pytest tests/test_repo_gateway.py tests/test_repo_governance.py tests/test_repo_links.py tests/test_repo_registry.py tests/test_repo_triage.py tests/test_observability.py`
    - Result: `16 passed in 0.09s`
  - `python3 -m py_compile src/bigclaw/*.py`
    - Result: passed with no output
  - `find src/bigclaw -maxdepth 1 -name '*.py' | wc -l`
    - Result: `13`
  - `PYTHONPATH=src python3 - <<'PY' ... importlib.import_module(...) ... PY`
    - Result: legacy workspace bootstrap imports resolved successfully:
      `bigclaw.workspace_bootstrap -> bigclaw.__main__`
      `bigclaw.workspace_bootstrap_validation -> bigclaw.__main__`
      `bigclaw.workspace_bootstrap_cli -> bigclaw.__main__`
      `bigclaw.github_sync -> bigclaw.__main__`
      `bigclaw.__main__ -> bigclaw.__main__`
  - `PYTHONPATH=src python3 -m pytest tests/test_workspace_bootstrap.py tests/test_github_sync.py tests/test_scheduler.py tests/test_workflow.py tests/test_operations.py`
    - Result: `46 passed in 4.18s`
  - `python3 -m py_compile src/bigclaw/*.py`
    - Result: passed with no output
  - `find src/bigclaw -maxdepth 1 -name '*.py' | wc -l`
    - Result: `12`
  - `PYTHONPATH=src python3 - <<'PY' ... importlib.import_module(...) ... PY`
    - Result: legacy workflow imports resolved successfully:
      `bigclaw.workflow -> bigclaw.scheduler`
      `bigclaw.scheduler -> bigclaw.scheduler`
      `bigclaw.dsl -> bigclaw.scheduler`
      `bigclaw -> bigclaw`
  - `PYTHONPATH=src python3 -m pytest tests/test_dsl.py tests/test_workflow.py tests/test_scheduler.py tests/test_audit_events.py`
    - Result: `21 passed in 0.10s`
  - `python3 -m py_compile src/bigclaw/*.py`
    - Result: passed with no output
  - `find src/bigclaw -maxdepth 1 -name '*.py' | wc -l`
    - Result: `11`
  - `PYTHONPATH=src python3 - <<'PY' ... importlib.import_module(...) ... PY`
    - Result: legacy model imports resolved successfully:
      `bigclaw.models -> bigclaw.execution_contract`
      `bigclaw.execution_contract -> bigclaw.execution_contract`
      `bigclaw.connectors -> bigclaw.execution_contract`
      `bigclaw.mapping -> bigclaw.execution_contract`
      `bigclaw -> bigclaw`
  - `PYTHONPATH=src python3 -m pytest tests/test_models.py tests/test_execution_contract.py tests/test_scheduler.py tests/test_reports.py tests/test_observability.py`
    - Result: `56 passed in 0.11s`
  - `python3 -m py_compile src/bigclaw/*.py`
    - Result: passed with no output
  - `find src/bigclaw -maxdepth 1 -name '*.py' | wc -l`
    - Result: `10`
  - `PYTHONPATH=src python3 - <<'PY' ... getattr(bigclaw, name) ... PY`
    - Result: legacy planning imports resolved successfully:
      `bigclaw.planning -> bigclaw.reports`
      `bigclaw.governance -> bigclaw.reports`
      `bigclaw.roadmap -> bigclaw.reports`
      `bigclaw.reports -> bigclaw.reports`
  - `PYTHONPATH=src python3 -m pytest tests/test_planning.py tests/test_governance.py tests/test_repo_rollout.py tests/test_reports.py`
    - Result: `54 passed in 0.10s`
  - `python3 -m py_compile src/bigclaw/*.py`
    - Result: passed with no output
  - `find src/bigclaw -maxdepth 1 -name '*.py' | wc -l`
    - Result: `9`
  - `PYTHONPATH=src python3 - <<'PY' ... getattr(bigclaw, name) ... PY`
    - Result: legacy UI review imports resolved successfully:
      `bigclaw.ui_review -> bigclaw.design_system`
      `bigclaw.design_system -> bigclaw.design_system`
      `bigclaw.console_ia -> bigclaw.design_system`
  - `PYTHONPATH=src python3 -m pytest tests/test_ui_review.py tests/test_design_system.py`
    - Result: `43 passed in 0.21s`
  - `python3 -m py_compile src/bigclaw/*.py`
    - Result: passed with no output
  - `find src/bigclaw -maxdepth 1 -name '*.py' | wc -l`
    - Result: `8`
