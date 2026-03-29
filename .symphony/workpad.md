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
- Directly handled deleted Python files in `src/bigclaw/`:
  - `deprecation.py`
  - `mapping.py`
  - `parallel_refill.py`
  - `cost_control.py`
  - `pilot.py`
  - `repo_commits.py`
  - `repo_gateway.py`
  - `repo_governance.py`
  - `repo_links.py`
  - `repo_registry.py`
  - `repo_triage.py`
  - `roadmap.py`
  - `validation_policy.py`
- Replacement / consolidation targets:
  - `legacy_shim.py` now owns the legacy runtime deprecation helpers.
  - `connectors.py` now owns source-issue mapping helpers.
  - `operations.py` now owns the budget control helpers.
  - `queue.py` now owns the parallel refill queue helpers.
  - `planning.py` now owns the execution-pack roadmap dataclasses and builder.
  - `repo_plane.py` now owns the repo commit, gateway, governance, link, registry, and triage surfaces.
  - `reports.py` now owns the pilot implementation and validation report policy helpers.
  - `__init__.py` now registers compatibility aliases so `import bigclaw.<old_module>` still resolves.
- Retained nearby Python files and reasons:
  - `execution_contract.py`: retained as the generic permission-contract host; repo policy compatibility now aliases into `repo_plane.py` without widening into broader contract semantics.
  - `reports.py`: retained as the primary reporting host after absorbing pilot and validation helpers; further consolidation there would stop being low-risk.
  - `operations.py`: retained as the operations-policy host after absorbing budget control helpers; broader merging beyond this would widen the issue.
- Python file count impact under `src/bigclaw/*.py`:
  - Before: `49`
  - After: `37`
  - Delta: `-12`
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
