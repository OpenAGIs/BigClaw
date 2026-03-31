Issue: BIG-GO-1030

Plan
- Merge `tests/test_orchestration.py` into `tests/test_runtime_matrix.py`.
- Remove the standalone orchestration regression file after its assertions live in the surviving runtime-oriented test module.
- Update directly coupled planning references so they point at `tests/test_runtime_matrix.py` instead of the deleted orchestration file.
- Re-run targeted pytest coverage for runtime/orchestration, recalculate repository `.py` / `.go` / `pyproject` / `setup` counts, then commit and push.

Acceptance
- The repository physical `.py` file count decreases again.
- `tests/test_orchestration.py` is removed from the tree.
- Orchestration coverage still passes from `tests/test_runtime_matrix.py`.
- Planning references no longer point at a deleted orchestration test file.
- Final report includes the exact impact on `.py` count, `.go` count, and `pyproject.toml` / `setup.py` / `setup.cfg` presence.

Validation
- `PYTHONPATH=src python3 -m pytest tests/test_runtime_matrix.py -q`
- `find . -type f \\( -name '*.py' -o -name '*.go' -o -name 'pyproject.toml' -o -name 'setup.py' -o -name 'setup.cfg' \\) | sed 's#^./##' | awk 'BEGIN{py=0;go=0;pp=0;setup=0} /\\.py$/{py++} /\\.go$/{go++} /pyproject\\.toml$/{pp++} /(setup\\.py|setup\\.cfg)$/{setup++} END{printf("py=%d\\ngo=%d\\npyproject=%d\\nsetup=%d\\n",py,go,pp,setup)}'`
- `git diff --stat && git status --short`
