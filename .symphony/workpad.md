Issue: BIG-GO-1030

Plan
- Fold `src/bigclaw/collaboration.py` into `src/bigclaw/observability.py`.
- Preserve compatibility for `bigclaw.collaboration` imports through the package legacy-surface shim in `src/bigclaw/__init__.py`.
- Re-run targeted observability and reports pytest coverage plus a direct import-compatibility check, recalculate repository `.py` / `.go` / `pyproject` / `setup` counts, then commit and push.

Acceptance
- The repository physical `.py` file count decreases again.
- `src/bigclaw/collaboration.py` is removed from the tree.
- Collaboration-building and report rendering behaviors still pass from `src/bigclaw/observability.py` / `src/bigclaw/reports.py` and their targeted tests.
- `from bigclaw.collaboration import ...` continues to resolve through the compatibility surface.
- Final report includes the exact impact on `.py` count, `.go` count, and `pyproject.toml` / `setup.py` / `setup.cfg` presence.

Validation
- `PYTHONPATH=src python3 -m pytest tests/test_observability.py tests/test_reports.py -q`
- `PYTHONPATH=src python3 - <<'PY'\nfrom bigclaw.collaboration import CollaborationThread, build_collaboration_thread\nprint(CollaborationThread.__name__)\nprint(build_collaboration_thread.__name__)\nPY`
- `find . -type f \( -name '*.py' -o -name '*.go' -o -name 'pyproject.toml' -o -name 'setup.py' -o -name 'setup.cfg' \) | sed 's#^./##' | awk 'BEGIN{py=0;go=0;pp=0;setup=0} /\.py$/{py++} /\.go$/{go++} /pyproject\.toml$/{pp++} /(setup\.py|setup\.cfg)$/{setup++} END{printf("py=%d\ngo=%d\npyproject=%d\nsetup=%d\n",py,go,pp,setup)}'`
- `git diff --stat && git status --short`
