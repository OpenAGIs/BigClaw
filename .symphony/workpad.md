Issue: BIG-GO-1030

Plan
- Fold `src/bigclaw/planning.py` into `src/bigclaw/reports.py`.
- Preserve compatibility for `bigclaw.planning`, `bigclaw.governance`, and `bigclaw.workspace_bootstrap` imports through the package legacy-surface shim in `src/bigclaw/__init__.py`.
- Re-run targeted planning pytest coverage plus direct import-compatibility checks, recalculate repository `.py` / `.go` / `pyproject` / `setup` counts, then commit and push.

Acceptance
- The repository physical `.py` file count decreases again.
- `src/bigclaw/planning.py` is removed from the tree.
- Planning, governance, and workspace-bootstrap behaviors still pass from `src/bigclaw/reports.py` and the targeted tests.
- `from bigclaw.planning import ...`, `from bigclaw.governance import ...`, and `from bigclaw.workspace_bootstrap import ...` continue to resolve through the compatibility surface.
- Final report includes the exact impact on `.py` count, `.go` count, and `pyproject.toml` / `setup.py` / `setup.cfg` presence.

Validation
- `PYTHONPATH=src python3 -m pytest tests/test_planning.py -q`
- `PYTHONPATH=src python3 - <<'PY'\nfrom bigclaw.planning import CandidateBacklog, WorkspaceBootstrapStatus\nfrom bigclaw.workspace_bootstrap import bootstrap_workspace\nfrom bigclaw.governance import ScopeFreezeBoard\nprint(CandidateBacklog.__name__)\nprint(WorkspaceBootstrapStatus.__name__)\nprint(bootstrap_workspace.__name__)\nprint(ScopeFreezeBoard.__name__)\nPY`
- `find . -type f \( -name '*.py' -o -name '*.go' -o -name 'pyproject.toml' -o -name 'setup.py' -o -name 'setup.cfg' \) | sed 's#^./##' | awk 'BEGIN{py=0;go=0;pp=0;setup=0} /\.py$/{py++} /\.go$/{go++} /pyproject\.toml$/{pp++} /(setup\.py|setup\.cfg)$/{setup++} END{printf("py=%d\ngo=%d\npyproject=%d\nsetup=%d\n",py,go,pp,setup)}'`
- `git diff --stat && git status --short`
