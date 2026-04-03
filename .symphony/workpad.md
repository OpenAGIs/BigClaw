Issue: BIG-GO-1030

Plan
- Fold `src/bigclaw/design_system.py` into `src/bigclaw/reports.py`.
- Preserve compatibility for `bigclaw.design_system` and `bigclaw.console_ia` imports through the package legacy-surface shim in `src/bigclaw/__init__.py`.
- Re-run targeted design-system pytest coverage plus direct import-compatibility checks, recalculate repository `.py` / `.go` / `pyproject` / `setup` counts, then commit and push.

Acceptance
- The repository physical `.py` file count decreases again.
- `src/bigclaw/design_system.py` is removed from the tree.
- Design-system and console-IA behaviors still pass from `src/bigclaw/reports.py` and `tests/test_design_system.py`.
- `from bigclaw.design_system import ...` and `from bigclaw.console_ia import ...` continue to resolve through the compatibility surface.
- Final report includes the exact impact on `.py` count, `.go` count, and `pyproject.toml` / `setup.py` / `setup.cfg` presence.

Validation
- `PYTHONPATH=src python3 -m pytest tests/test_design_system.py -q`
- `PYTHONPATH=src python3 - <<'PY'\nfrom bigclaw.design_system import DesignSystem, render_design_system_report\nfrom bigclaw.console_ia import ConsoleIAAuditor, render_console_ia_report\nprint(DesignSystem.__name__)\nprint(render_design_system_report.__name__)\nprint(ConsoleIAAuditor.__name__)\nprint(render_console_ia_report.__name__)\nPY`
- `find . -type f \( -name '*.py' -o -name '*.go' -o -name 'pyproject.toml' -o -name 'setup.py' -o -name 'setup.cfg' \) | sed 's#^./##' | awk 'BEGIN{py=0;go=0;pp=0;setup=0} /\.py$/{py++} /\.go$/{go++} /pyproject\.toml$/{pp++} /(setup\.py|setup\.cfg)$/{setup++} END{printf("py=%d\ngo=%d\npyproject=%d\nsetup=%d\n",py,go,pp,setup)}'`
- `git diff --stat && git status --short`
