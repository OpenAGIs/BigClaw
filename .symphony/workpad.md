Issue: BIG-GO-1024

Plan
- Finalize the current `src/bigclaw` tranche by keeping `audit_events`, `saved_views`, and `collaboration` exposed through dynamic support submodules inside `src/bigclaw/__init__.py` while the standalone `.py` files stay removed.
- Update any remaining repo references that still point at deleted `src/bigclaw/*.py` paths, limiting scope to direct fallout from this tranche.
- Run targeted smoke/tests plus file-count checks, then commit and push the issue branch.

Acceptance
- `src/bigclaw/audit_events.py`, `src/bigclaw/saved_views.py`, and `src/bigclaw/collaboration.py` remain deleted, reducing physical Python file count in `src/bigclaw`.
- `bigclaw.audit_events`, `bigclaw.saved_views`, and `bigclaw.collaboration` remain importable and preserve the tested behavior used by runtime, reports, observability, and planning.
- Static evidence or planning references in this slice do not point at deleted support-module file paths.
- No `pyproject.toml`, `setup.py`, or `setup.cfg` changes are introduced unless directly required by this tranche.

Validation
- `find src/bigclaw -maxdepth 1 -name '*.py' | sort | wc -l`
- `find . -name '*.py' | wc -l`
- `find . -name '*.go' | wc -l`
- `python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/planning.py tests/test_planning.py`
- `PYTHONPATH=src python3 -m pytest tests/test_audit_events.py tests/test_saved_views.py tests/test_repo_collaboration.py tests/test_observability.py tests/test_reports.py tests/test_planning.py -q`
- `PYTHONPATH=src python3 -m pytest tests/test_legacy_surface_modules.py -q`
- `PYTHONPATH=src python3 - <<'PY'`
- `import bigclaw.audit_events, bigclaw.saved_views, bigclaw.collaboration`
- `print("module smoke checks passed")`
- `PY`
- `git status --short`
- `git diff --stat`
