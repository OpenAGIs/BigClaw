Issue: BIG-GO-1030

Plan
- Fold `tests/test_design_system.py`, `tests/test_ui_review.py`, `tests/test_operations.py`, and `tests/test_planning.py` into `tests/test_reports.py`, and fold `tests/test_runtime_matrix.py` into `tests/test_observability.py`.
- Preserve the existing report, planning, operations, runtime, and observability assertions while removing standalone compatibility-layer test files.
- Re-run targeted reports and observability pytest coverage, recalculate repository `.py` / `.go` / `pyproject` / `setup` counts, then commit and push.

Acceptance
- The repository physical `.py` file count decreases again.
- `tests/test_design_system.py`, `tests/test_ui_review.py`, `tests/test_operations.py`, `tests/test_planning.py`, and `tests/test_runtime_matrix.py` are removed from the tree.
- Design-system, console-IA, UI-review, planning, operations, runtime-matrix, and observability behaviors still pass from the consolidated `tests/test_reports.py` and `tests/test_observability.py` coverage.
- Final report includes the exact impact on `.py` count, `.go` count, and `pyproject.toml` / `setup.py` / `setup.cfg` presence.

Validation
- `PYTHONPATH=src python3 -m pytest tests/test_reports.py tests/test_observability.py -q`
- `find . -type f \( -name '*.py' -o -name '*.go' -o -name 'pyproject.toml' -o -name 'setup.py' -o -name 'setup.cfg' \) | sed 's#^./##' | awk 'BEGIN{py=0;go=0;pp=0;setup=0} /\.py$/{py++} /\.go$/{go++} /pyproject\.toml$/{pp++} /(setup\.py|setup\.cfg)$/{setup++} END{printf("py=%d\ngo=%d\npyproject=%d\nsetup=%d\n",py,go,pp,setup)}'`
- `git diff --stat && git status --short`
