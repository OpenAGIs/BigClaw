Issue: BIG-GO-1014

Plan
- Inspect residual `src/bigclaw/**` Python modules and choose one low-risk consolidation target that reduces file count without widening scope.
- Fold `src/bigclaw/planning.py` into an existing retained module, then expose a compatibility `bigclaw.planning` surface from `src/bigclaw/__init__.py`.
- Remove the standalone `planning.py` file and update package wiring so existing imports and tests continue to work.
- Run targeted tests around planning, governance, rollout, and package import compatibility.
- Record file-count/build-file impact, then commit and push the branch.

Acceptance
- Repository result directly reduces residual Python assets under `src/bigclaw/**`.
- `.py` file count decreases while preserving `bigclaw.planning` import compatibility.
- Final report includes impact on `py files`, `go files`, `pyproject.toml`, and `setup.py`.
- Validation evidence includes exact commands and results.

Validation
- `python3 -m pytest tests/test_planning.py tests/test_repo_rollout.py tests/test_governance.py`
- `python3 -m pytest tests/test_models.py`
- `python3 - <<'PY'` import check for `bigclaw.planning` synthetic module
- `rg --files src/bigclaw -g '*.py' -g '*.go'`

Continuation 2

Plan
- Fold `src/bigclaw/operations.py` into `src/bigclaw/reports.py`, following the same residual-surface consolidation pattern used for planning.
- Rewire `src/bigclaw/__init__.py` so `bigclaw.operations` remains import-compatible as a synthetic module backed by `reports.py`.
- Preserve the existing `bigclaw.evaluation` compatibility surface after the move.
- Run focused validation for operations, control center, evaluation, and reports paths.
- Record updated inventory impact, then commit and push the continuation.

Acceptance
- `src/bigclaw/*.py` count decreases again.
- `bigclaw.operations` and `bigclaw.evaluation` imports remain functional after removing the standalone module.
- Final report still includes exact impacts on `py files`, `go files`, `pyproject.toml`, and `setup.py`.

Validation
- `PYTHONPATH=src python3 - <<'PY'` import checks for `bigclaw.operations` and `bigclaw.evaluation`
- `PYTHONPATH=src python3 -m py_compile src/bigclaw/reports.py src/bigclaw/__init__.py src/bigclaw/models.py src/bigclaw/runtime.py src/bigclaw/observability.py src/bigclaw/repository.py`
- `PYTHONPATH=src python3 -m pytest tests/test_operations.py tests/test_control_center.py tests/test_evaluation.py tests/test_reports.py`
- `rg --files src/bigclaw -g '*.py' -g '*.go'`

Continuation 3

Plan
- Fold the residual `src/bigclaw/reports.py` surface into `src/bigclaw/runtime.py`.
- Keep `bigclaw.reports`, `bigclaw.operations`, `bigclaw.evaluation`, `bigclaw.run_detail`, `bigclaw.pilot`, and `bigclaw.validation_policy` import-compatible via `src/bigclaw/__init__.py`.
- Remove the standalone `reports.py` file and revalidate the package surfaces that depended on it.
- Run focused report/runtime/operations validation and record updated inventory impact.
- Commit and push the continuation once the moved report surfaces are stable.

Acceptance
- `src/bigclaw/*.py` count decreases again.
- Existing package imports for the moved reporting surfaces still resolve.
- Final report continues to include exact `py files` / `go files` / `pyproject.toml` / `setup.py` impact.

Validation
- `PYTHONPATH=src python3 - <<'PY'` import checks for `bigclaw.reports`, `bigclaw.operations`, and `bigclaw.run_detail`
- `PYTHONPATH=src python3 -m py_compile src/bigclaw/runtime.py src/bigclaw/__init__.py src/bigclaw/models.py src/bigclaw/observability.py src/bigclaw/repository.py`
- `PYTHONPATH=src python3 -m pytest tests/test_reports.py tests/test_operations.py tests/test_control_center.py tests/test_evaluation.py tests/test_runtime_matrix.py`
- `rg --files src/bigclaw -g '*.py' -g '*.go'`

Continuation 4

Plan
- Fold the residual `src/bigclaw/observability.py` implementation into `src/bigclaw/runtime.py`.
- Rewire `src/bigclaw/__init__.py` so `bigclaw.observability`, `bigclaw.audit_events`, and `bigclaw.event_bus` stay import-compatible from the merged runtime surface.
- Remove the standalone `observability.py` file and keep repository/report/runtime imports coherent through the synthetic module.
- Run focused observability, reports, audit-event, event-bus, evaluation, and runtime validation.
- Record updated inventory counts, then commit and push if the merged surface is stable.

Acceptance
- `src/bigclaw/*.py` count decreases again.
- Existing observability-related import paths still resolve after removing the standalone module.
- Final report still includes exact impacts on `py files`, `go files`, `pyproject.toml`, and `setup.py`.

Validation
- `PYTHONPATH=src python3 - <<'PY'` import checks for `bigclaw.observability`, `bigclaw.audit_events`, and `bigclaw.event_bus`
- `PYTHONPATH=src python3 -m py_compile src/bigclaw/runtime.py src/bigclaw/__init__.py src/bigclaw/models.py src/bigclaw/repository.py`
- `PYTHONPATH=src python3 -m pytest tests/test_observability.py tests/test_audit_events.py tests/test_event_bus.py tests/test_reports.py tests/test_evaluation.py tests/test_runtime_matrix.py`
- `rg --files src/bigclaw -g '*.py' -g '*.go'`

Continuation 5

Plan
- Fold the residual `src/bigclaw/repository.py` implementation into `src/bigclaw/runtime.py`.
- Rewire `src/bigclaw/__init__.py` so `bigclaw.repository` and all repository-backed synthetic submodules continue to resolve from the merged runtime surface.
- Remove the standalone `repository.py` file and keep runtime/repository import compatibility stable.
- Run focused repository, runtime, observability, and report-facing validation against the merged surface.
- Record updated inventory counts, then commit and push if the merged package remains coherent.

Acceptance
- `src/bigclaw/*.py` count decreases again.
- `bigclaw.repository` and repository-backed package surfaces still resolve after removing the standalone module.
- Final report continues to include exact `py files`, `go files`, `pyproject.toml`, and `setup.py` impact.

Validation
- `PYTHONPATH=src python3 - <<'PY'` import checks for `bigclaw.repository`, `bigclaw.repo_registry`, and `bigclaw.repo_gateway`
- `PYTHONPATH=src python3 -m py_compile src/bigclaw/runtime.py src/bigclaw/__init__.py src/bigclaw/models.py`
- `PYTHONPATH=src python3 -m pytest tests/test_repo_registry.py tests/test_repo_gateway.py tests/test_repo_links.py tests/test_repo_board.py tests/test_repo_governance.py tests/test_repo_triage.py tests/test_repo_collaboration.py tests/test_reports.py tests/test_observability.py tests/test_runtime_matrix.py`
- `rg --files src/bigclaw -g '*.py' -g '*.go'`

Continuation 6

Plan
- Fold the residual `src/bigclaw/models.py` implementation into `src/bigclaw/runtime.py`.
- Rewire `src/bigclaw/__init__.py` so `bigclaw.models` and all model-backed compatibility surfaces resolve from the merged runtime surface.
- Remove the standalone `models.py` file, leaving only the core package init plus one merged implementation module under `src/bigclaw`.
- Run focused models/planning/risk/runtime/package-surface validation to catch any import-order regressions.
- Record updated inventory counts, then commit and push if the two-file package layout is stable.

Acceptance
- `src/bigclaw/*.py` count decreases again.
- `bigclaw.models` and model-backed compatibility imports still resolve after removing the standalone module.
- Final report continues to include exact `py files`, `go files`, `pyproject.toml`, and `setup.py` impact.

Validation
- `PYTHONPATH=src python3 - <<'PY'` import checks for `bigclaw.models`, `bigclaw.planning`, and `bigclaw.risk`
- `PYTHONPATH=src python3 -m py_compile src/bigclaw/runtime.py src/bigclaw/__init__.py`
- `PYTHONPATH=src python3 -m pytest tests/test_models.py tests/test_planning.py tests/test_risk.py tests/test_runtime_matrix.py tests/test_memory.py tests/test_repo_rollout.py tests/test_governance.py tests/test_dsl.py`
- `rg --files src/bigclaw -g '*.py' -g '*.go'`

Continuation 7

Plan
- Fold the residual `src/bigclaw/runtime.py` implementation into `src/bigclaw/__init__.py`.
- Rewire `bigclaw.runtime` to remain import-compatible as a synthetic module backed by the package module itself.
- Remove the standalone `runtime.py` file, leaving a single Python source file under `src/bigclaw`.
- Run focused package-surface validation across runtime, repository, reports, observability, models, and planning compatibility imports.
- Record final inventory counts, then commit and push if the single-file package layout is stable.

Acceptance
- `src/bigclaw/*.py` count decreases again.
- `bigclaw.runtime` and all package compatibility imports still resolve after removing the standalone module.
- Final report continues to include exact `py files`, `go files`, `pyproject.toml`, and `setup.py` impact.

Validation
- `PYTHONPATH=src python3 - <<'PY'` import checks for `bigclaw.runtime`, `bigclaw.repository`, `bigclaw.models`, `bigclaw.reports`, and `bigclaw.observability`
- `PYTHONPATH=src python3 -m py_compile src/bigclaw/__init__.py`
- `PYTHONPATH=src python3 -m pytest tests/test_runtime_matrix.py tests/test_reports.py tests/test_observability.py tests/test_models.py tests/test_planning.py tests/test_repo_gateway.py tests/test_repo_registry.py tests/test_control_center.py tests/test_evaluation.py`
- `rg --files src/bigclaw -g '*.py' -g '*.go'`
