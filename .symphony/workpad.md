Issue: BIG-GO-1030

Plan
- Fold `src/bigclaw/workspace_bootstrap.py` into `src/bigclaw/planning.py`.
- Preserve compatibility for `bigclaw.workspace_bootstrap` imports through the package legacy-surface shim in `src/bigclaw/__init__.py`.
- Re-run targeted planning pytest coverage plus a direct import-compatibility check, recalculate repository `.py` / `.go` / `pyproject` / `setup` counts, then commit and push.

Acceptance
- The repository physical `.py` file count decreases again.
- `src/bigclaw/workspace_bootstrap.py` is removed from the tree.
- Workspace bootstrap behaviors still pass from `src/bigclaw/planning.py` / `tests/test_planning.py`.
- `from bigclaw.workspace_bootstrap import ...` continues to resolve through the compatibility surface.
- Final report includes the exact impact on `.py` count, `.go` count, and `pyproject.toml` / `setup.py` / `setup.cfg` presence.

Validation
- `PYTHONPATH=src python3 -m pytest tests/test_planning.py -q`
- `PYTHONPATH=src python3 - <<'PY'\nfrom bigclaw.workspace_bootstrap import bootstrap_workspace, cleanup_workspace\nprint(bootstrap_workspace.__name__)\nprint(cleanup_workspace.__name__)\nPY`
- `find . -type f \( -name '*.py' -o -name '*.go' -o -name 'pyproject.toml' -o -name 'setup.py' -o -name 'setup.cfg' \) | sed 's#^./##' | awk 'BEGIN{py=0;go=0;pp=0;setup=0} /\.py$/{py++} /\.go$/{go++} /pyproject\.toml$/{pp++} /(setup\.py|setup\.cfg)$/{setup++} END{printf("py=%d\ngo=%d\npyproject=%d\nsetup=%d\n",py,go,pp,setup)}'`
- `git diff --stat && git status --short`
