Issue: BIG-GO-1024

Plan
- Fold `repo_plane.py` and `repo_links.py` into `src/bigclaw/__init__.py` as dynamic support submodules while preserving `bigclaw.repo_plane` and `bigclaw.repo_links` imports.
- Delete those standalone Python files from `src/bigclaw`.
- Run targeted repo-plane/repo-links validation plus file-count checks, then commit and push the issue branch.

Acceptance
- `src/bigclaw/repo_plane.py` and `src/bigclaw/repo_links.py` are removed, reducing physical Python file count in `src/bigclaw`.
- `bigclaw.repo_plane` and `bigclaw.repo_links` remain importable and preserve the tested behavior used by observability and repo registry flows.
- No `pyproject.toml`, `setup.py`, or `setup.cfg` changes are introduced unless directly required by this tranche.

Validation
- `find src/bigclaw -maxdepth 1 -name '*.py' | sort | wc -l`
- `find . -name '*.py' | wc -l`
- `find . -name '*.go' | wc -l`
- `python3 -m py_compile src/bigclaw/__init__.py tests/test_repo_links.py tests/test_repo_registry.py`
- `PYTHONPATH=src python3 -m pytest tests/test_repo_links.py tests/test_repo_registry.py tests/test_repo_board.py tests/test_observability.py -q`
- `PYTHONPATH=src python3 - <<'PY'`
- `import bigclaw.repo_plane, bigclaw.repo_links`
- `print("module smoke checks passed")`
- `PY`
- `git status --short`
- `git diff --stat`
