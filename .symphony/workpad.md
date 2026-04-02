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
