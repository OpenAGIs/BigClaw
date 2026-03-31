Issue: BIG-GO-1030

Plan
- Merge the thin scheduler regression file into `tests/test_runtime_matrix.py` and delete `tests/test_scheduler.py`.
- Merge the queue control-center regression file into `tests/test_operations.py` and delete `tests/test_control_center.py`.
- Update directly coupled planning evidence/validation references so they point at the surviving merged test files instead of deleted paths.
- Re-run targeted pytest slices for runtime and operations/reporting coverage, recalculate repository `.py` / `.go` / `pyproject` / `setup` counts, then commit and push.

Acceptance
- The repository physical `.py` file count decreases again.
- `tests/test_scheduler.py` and `tests/test_control_center.py` are removed from the tree.
- Merged scheduler and control-center coverage still passes from the surviving test files.
- Planning references no longer point at deleted test files.
- Final report includes the exact impact on `.py` count, `.go` count, and `pyproject.toml` / `setup.py` / `setup.cfg` presence.

Validation
- `PYTHONPATH=src python3 -m pytest tests/test_runtime_matrix.py tests/test_operations.py -q`
- `find . -type f \\( -name '*.py' -o -name '*.go' -o -name 'pyproject.toml' -o -name 'setup.py' -o -name 'setup.cfg' \\) | sed 's#^./##' | awk 'BEGIN{py=0;go=0;pp=0;setup=0} /\\.py$/{py++} /\\.go$/{go++} /pyproject\\.toml$/{pp++} /(setup\\.py|setup\\.cfg)$/{setup++} END{printf("py=%d\\ngo=%d\\npyproject=%d\\nsetup=%d\\n",py,go,pp,setup)}'`
- `git diff --stat && git status --short`
