Issue: BIG-GO-1030

Plan
- Merge `tests/test_console_ia.py` into `tests/test_design_system.py`.
- Remove the standalone console-IA regression file after its assertions live in the surviving design-system test module.
- Update directly coupled planning references so the release-control candidate points at `tests/test_design_system.py` and `tests/test_ui_review.py`.
- Re-run targeted pytest coverage for design-system and planning surfaces, recalculate repository `.py` / `.go` / `pyproject` / `setup` counts, then commit and push.

Acceptance
- The repository physical `.py` file count decreases again.
- `tests/test_console_ia.py` is removed from the tree.
- Console IA coverage still passes from `tests/test_design_system.py`.
- Planning references no longer point at the deleted console-IA test file.
- Final report includes the exact impact on `.py` count, `.go` count, and `pyproject.toml` / `setup.py` / `setup.cfg` presence.

Validation
- `PYTHONPATH=src python3 -m pytest tests/test_design_system.py tests/test_planning.py -q`
- `find . -type f \\( -name '*.py' -o -name '*.go' -o -name 'pyproject.toml' -o -name 'setup.py' -o -name 'setup.cfg' \\) | sed 's#^./##' | awk 'BEGIN{py=0;go=0;pp=0;setup=0} /\\.py$/{py++} /\\.go$/{go++} /pyproject\\.toml$/{pp++} /(setup\\.py|setup\\.cfg)$/{setup++} END{printf("py=%d\\ngo=%d\\npyproject=%d\\nsetup=%d\\n",py,go,pp,setup)}'`
- `git diff --stat && git status --short`
