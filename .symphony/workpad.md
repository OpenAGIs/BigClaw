Issue: BIG-GO-1030

Plan
- Replace the remaining Python test asset `tests/test_reports.py` with a non-Python smoke harness.
- Preserve targeted validation for report, observability, runtime, planning, and operations surfaces through a shell runner that executes inline Python assertions.
- Re-run the shell harness, recalculate repository `.py` / `.go` / `pyproject` / `setup` counts, then commit and push.

Acceptance
- The repository physical `.py` file count decreases again.
- `tests/test_reports.py` is removed from the tree.
- Report, planning, operations, runtime, workflow, orchestration, and observability behaviors still pass through the replacement smoke harness.
- Final report includes the exact impact on `.py` count, `.go` count, and `pyproject.toml` / `setup.py` / `setup.cfg` presence.

Validation
- `bash tests/report_surface_smoke.sh`
- `find . -type f \( -name '*.py' -o -name '*.go' -o -name 'pyproject.toml' -o -name 'setup.py' -o -name 'setup.cfg' \) | sed 's#^./##' | awk 'BEGIN{py=0;go=0;pp=0;setup=0} /\.py$/{py++} /\.go$/{go++} /pyproject\.toml$/{pp++} /(setup\.py|setup\.cfg)$/{setup++} END{printf("py=%d\ngo=%d\npyproject=%d\nsetup=%d\n",py,go,pp,setup)}'`
- `git diff --stat && git status --short`
