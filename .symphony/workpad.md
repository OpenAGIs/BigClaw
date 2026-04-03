Issue: BIG-GO-1030

Plan
- Fold `src/bigclaw/models.py` into `src/bigclaw/observability.py`.
- Preserve compatibility for `bigclaw.models`, `bigclaw.connectors`, and `bigclaw.dsl` imports through the package legacy-surface shim in `src/bigclaw/__init__.py`.
- Re-run targeted runtime-matrix, observability, and operations pytest coverage plus direct import-compatibility checks, recalculate repository `.py` / `.go` / `pyproject` / `setup` counts, then commit and push.

Acceptance
- The repository physical `.py` file count decreases again.
- `src/bigclaw/models.py` is removed from the tree.
- Model, connector, and workflow-definition behaviors still pass from `src/bigclaw/observability.py` and the targeted tests.
- `from bigclaw.models import ...`, `from bigclaw.connectors import ...`, and `from bigclaw.dsl import ...` continue to resolve through the compatibility surface.
- Final report includes the exact impact on `.py` count, `.go` count, and `pyproject.toml` / `setup.py` / `setup.cfg` presence.

Validation
- `PYTHONPATH=src python3 -m pytest tests/test_runtime_matrix.py tests/test_observability.py tests/test_operations.py -q`
- `PYTHONPATH=src python3 - <<'PY'\nfrom bigclaw.models import Task, WorkflowDefinition\nfrom bigclaw.connectors import GitHubConnector\nprint(Task.__name__)\nprint(WorkflowDefinition.__name__)\nprint(GitHubConnector.__name__)\nPY`
- `find . -type f \( -name '*.py' -o -name '*.go' -o -name 'pyproject.toml' -o -name 'setup.py' -o -name 'setup.cfg' \) | sed 's#^./##' | awk 'BEGIN{py=0;go=0;pp=0;setup=0} /\.py$/{py++} /\.go$/{go++} /pyproject\.toml$/{pp++} /(setup\.py|setup\.cfg)$/{setup++} END{printf("py=%d\ngo=%d\npyproject=%d\nsetup=%d\n",py,go,pp,setup)}'`
- `git diff --stat && git status --short`
