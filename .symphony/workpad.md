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
