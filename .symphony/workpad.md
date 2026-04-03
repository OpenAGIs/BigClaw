Issue: BIG-GO-1030

Plan
- Fold `src/bigclaw/operations.py` into `src/bigclaw/reports.py`.
- Preserve compatibility for `bigclaw.operations`, `bigclaw.evaluation`, and `bigclaw.execution_contract` imports through the package legacy-surface shim in `src/bigclaw/__init__.py`.
- Re-run targeted operations and reports pytest coverage plus direct import-compatibility checks, recalculate repository `.py` / `.go` / `pyproject` / `setup` counts, then commit and push.

Acceptance
- The repository physical `.py` file count decreases again.
- `src/bigclaw/operations.py` is removed from the tree.
- Operations, evaluation, and execution-contract behaviors still pass from `src/bigclaw/reports.py` and the targeted tests.
- `from bigclaw.operations import ...`, `from bigclaw.evaluation import ...`, and `from bigclaw.execution_contract import ...` continue to resolve through the compatibility surface.
- Final report includes the exact impact on `.py` count, `.go` count, and `pyproject.toml` / `setup.py` / `setup.cfg` presence.

Validation
- `PYTHONPATH=src python3 -m pytest tests/test_operations.py tests/test_reports.py -q`
- `PYTHONPATH=src python3 - <<'PY'\nfrom bigclaw.operations import OperationsAnalytics, render_operations_dashboard\nfrom bigclaw.evaluation import BenchmarkRunner\nfrom bigclaw.execution_contract import ExecutionContract\nprint(OperationsAnalytics.__name__)\nprint(render_operations_dashboard.__name__)\nprint(BenchmarkRunner.__name__)\nprint(ExecutionContract.__name__)\nPY`
- `find . -type f \( -name '*.py' -o -name '*.go' -o -name 'pyproject.toml' -o -name 'setup.py' -o -name 'setup.cfg' \) | sed 's#^./##' | awk 'BEGIN{py=0;go=0;pp=0;setup=0} /\.py$/{py++} /\.go$/{go++} /pyproject\.toml$/{pp++} /(setup\.py|setup\.cfg)$/{setup++} END{printf("py=%d\ngo=%d\npyproject=%d\nsetup=%d\n",py,go,pp,setup)}'`
- `git diff --stat && git status --short`
