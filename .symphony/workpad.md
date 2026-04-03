Issue: BIG-GO-1030

Plan
- Fold `src/bigclaw/runtime.py` into `src/bigclaw/observability.py`.
- Preserve compatibility for `bigclaw.runtime`, `bigclaw.queue`, `bigclaw.scheduler`, `bigclaw.workflow`, `bigclaw.orchestration`, `bigclaw.risk`, and `bigclaw.service` imports through the package legacy-surface shim in `src/bigclaw/__init__.py`.
- Re-run targeted runtime-matrix and observability pytest coverage plus direct import-compatibility checks, recalculate repository `.py` / `.go` / `pyproject` / `setup` counts, then commit and push.

Acceptance
- The repository physical `.py` file count decreases again.
- `src/bigclaw/runtime.py` is removed from the tree.
- Runtime, scheduler, queue, workflow, orchestration, risk, and service behaviors still pass from `src/bigclaw/observability.py` and the targeted tests.
- `from bigclaw.runtime import ...`, `from bigclaw.queue import ...`, `from bigclaw.scheduler import ...`, `from bigclaw.workflow import ...`, `from bigclaw.orchestration import ...`, `from bigclaw.risk import ...`, and `from bigclaw.service import ...` continue to resolve through the compatibility surface.
- Final report includes the exact impact on `.py` count, `.go` count, and `pyproject.toml` / `setup.py` / `setup.cfg` presence.

Validation
- `PYTHONPATH=src python3 -m pytest tests/test_runtime_matrix.py tests/test_observability.py -q`
- `PYTHONPATH=src python3 - <<'PY'\nfrom bigclaw.runtime import ClawWorkerRuntime, ToolRuntime\nfrom bigclaw.scheduler import Scheduler\nfrom bigclaw.workflow import WorkflowEngine\nfrom bigclaw.orchestration import CrossDepartmentOrchestrator\nprint(ClawWorkerRuntime.__name__)\nprint(ToolRuntime.__name__)\nprint(Scheduler.__name__)\nprint(WorkflowEngine.__name__)\nprint(CrossDepartmentOrchestrator.__name__)\nPY`
- `find . -type f \( -name '*.py' -o -name '*.go' -o -name 'pyproject.toml' -o -name 'setup.py' -o -name 'setup.cfg' \) | sed 's#^./##' | awk 'BEGIN{py=0;go=0;pp=0;setup=0} /\.py$/{py++} /\.go$/{go++} /pyproject\.toml$/{pp++} /(setup\.py|setup\.cfg)$/{setup++} END{printf("py=%d\ngo=%d\npyproject=%d\nsetup=%d\n",py,go,pp,setup)}'`
- `git diff --stat && git status --short`
