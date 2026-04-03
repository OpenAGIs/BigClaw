Issue: BIG-GO-1030

Plan
- Fold `tests/test_observability.py` and `tests/conftest.py` into `tests/test_reports.py`.
- Preserve the existing consolidated report and observability assertions while removing the remaining standalone test files.
- Re-run targeted `tests/test_reports.py` coverage directly with `PYTHONPATH=src`, recalculate repository `.py` / `.go` / `pyproject` / `setup` counts, then commit and push.

Acceptance
- The repository physical `.py` file count decreases again.
- `tests/test_observability.py` and `tests/conftest.py` are removed from the tree.
- Report, planning, operations, runtime-matrix, UI-review, design-system, and observability behaviors still pass from the consolidated `tests/test_reports.py` coverage.
- Final report includes the exact impact on `.py` count, `.go` count, and `pyproject.toml` / `setup.py` / `setup.cfg` presence.

Validation
- `PYTHONPATH=src python3 -m pytest tests/test_reports.py -q`
- `find . -type f \( -name '*.py' -o -name '*.go' -o -name 'pyproject.toml' -o -name 'setup.py' -o -name 'setup.cfg' \) | sed 's#^./##' | awk 'BEGIN{py=0;go=0;pp=0;setup=0} /\.py$/{py++} /\.go$/{go++} /pyproject\.toml$/{pp++} /(setup\.py|setup\.cfg)$/{setup++} END{printf("py=%d\ngo=%d\npyproject=%d\nsetup=%d\n",py,go,pp,setup)}'`
- `git diff --stat && git status --short`
