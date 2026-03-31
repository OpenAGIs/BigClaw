Issue: BIG-GO-1030

Plan
- Merge `tests/test_workspace_bootstrap.py` into `tests/test_planning.py`.
- Remove the standalone workspace-bootstrap regression file after its assertions live in the surviving planning test module.
- Update directly coupled README smoke guidance so it points at the surviving merged test file.
- Re-run targeted pytest coverage for planning and workspace bootstrap, recalculate repository `.py` / `.go` / `pyproject` / `setup` counts, then commit and push.

Acceptance
- The repository physical `.py` file count decreases again.
- `tests/test_workspace_bootstrap.py` is removed from the tree.
- Workspace bootstrap coverage still passes from `tests/test_planning.py`.
- README smoke guidance no longer points at a deleted test file.
- Final report includes the exact impact on `.py` count, `.go` count, and `pyproject.toml` / `setup.py` / `setup.cfg` presence.

Validation
- `PYTHONPATH=src python3 -m pytest tests/test_planning.py -q`
- `find . -type f \\( -name '*.py' -o -name '*.go' -o -name 'pyproject.toml' -o -name 'setup.py' -o -name 'setup.cfg' \\) | sed 's#^./##' | awk 'BEGIN{py=0;go=0;pp=0;setup=0} /\\.py$/{py++} /\\.go$/{go++} /pyproject\\.toml$/{pp++} /(setup\\.py|setup\\.cfg)$/{setup++} END{printf("py=%d\\ngo=%d\\npyproject=%d\\nsetup=%d\\n",py,go,pp,setup)}'`
- `git diff --stat && git status --short`
