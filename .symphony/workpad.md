Issue: BIG-GO-1030

Plan
- Fold `src/bigclaw/reports.py` into `src/bigclaw/__init__.py`.
- Preserve compatibility for `bigclaw.reports`, `bigclaw.observability`, and the existing legacy surface modules through package-level shims and aliases in `src/bigclaw/__init__.py`.
- Re-run the shell harness, recalculate repository `.py` / `.go` / `pyproject` / `setup` counts, then commit and push.

Acceptance
- The repository physical `.py` file count decreases again.
- `src/bigclaw/reports.py` is removed from the tree.
- Report, planning, operations, runtime, workflow, orchestration, observability, and direct `bigclaw.reports` imports still pass through the replacement smoke harness.
- Final report includes the exact impact on `.py` count, `.go` count, and `pyproject.toml` / `setup.py` / `setup.cfg` presence.

Validation
- `bash tests/report_surface_smoke.sh`
- `PYTHONPATH=src python3 - <<'PY'\nfrom bigclaw.reports import ReportStudio, render_task_run_report\nfrom bigclaw.observability import ObservabilityLedger\nprint(ReportStudio.__name__)\nprint(render_task_run_report.__name__)\nprint(ObservabilityLedger.__name__)\nPY`
- `find . -type f \( -name '*.py' -o -name '*.go' -o -name 'pyproject.toml' -o -name 'setup.py' -o -name 'setup.cfg' \) | sed 's#^./##' | awk 'BEGIN{py=0;go=0;pp=0;setup=0} /\.py$/{py++} /\.go$/{go++} /pyproject\.toml$/{pp++} /(setup\.py|setup\.cfg)$/{setup++} END{printf("py=%d\ngo=%d\npyproject=%d\nsetup=%d\n",py,go,pp,setup)}'`
- `git diff --stat && git status --short`
