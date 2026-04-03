Issue: BIG-GO-1030

Plan
- Fold `src/bigclaw/observability.py` into `src/bigclaw/reports.py`.
- Preserve compatibility for `bigclaw.observability` and the existing report-facing legacy surfaces through the package shim in `src/bigclaw/__init__.py`.
- Re-run targeted `tests/test_reports.py` coverage directly with `PYTHONPATH=src`, recalculate repository `.py` / `.go` / `pyproject` / `setup` counts, then commit and push.

Acceptance
- The repository physical `.py` file count decreases again.
- `src/bigclaw/observability.py` is removed from the tree.
- Report, planning, operations, runtime, workflow, orchestration, and observability behaviors still pass from the consolidated `src/bigclaw/reports.py` and `tests/test_reports.py` coverage.
- Final report includes the exact impact on `.py` count, `.go` count, and `pyproject.toml` / `setup.py` / `setup.cfg` presence.

Validation
- `PYTHONPATH=src python3 -m pytest tests/test_reports.py -q`
- `PYTHONPATH=src python3 - <<'PY'\nfrom bigclaw.observability import ObservabilityLedger, TaskRun\nfrom bigclaw.reports import render_task_run_report\nprint(ObservabilityLedger.__name__)\nprint(TaskRun.__name__)\nprint(render_task_run_report.__name__)\nPY`
- `find . -type f \( -name '*.py' -o -name '*.go' -o -name 'pyproject.toml' -o -name 'setup.py' -o -name 'setup.cfg' \) | sed 's#^./##' | awk 'BEGIN{py=0;go=0;pp=0;setup=0} /\.py$/{py++} /\.go$/{go++} /pyproject\.toml$/{pp++} /(setup\.py|setup\.cfg)$/{setup++} END{printf("py=%d\ngo=%d\npyproject=%d\nsetup=%d\n",py,go,pp,setup)}'`
- `git diff --stat && git status --short`
