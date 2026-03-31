Issue: BIG-GO-1030

Plan
- Merge `tests/test_governance.py` into `tests/test_planning.py`.
- Merge `tests/test_live_shadow_bundle.py` into `tests/test_reports.py`.
- Remove the two standalone test files once their assertions live in the surviving modules.
- Re-run targeted pytest coverage for planning and reports, recalculate repository `.py` / `.go` / `pyproject` / `setup` counts, then commit and push.

Acceptance
- The repository physical `.py` file count decreases again.
- `tests/test_governance.py` and `tests/test_live_shadow_bundle.py` are removed from the tree.
- Governance and live-shadow artifact shape coverage still passes from the merged surviving test files.
- Final report includes the exact impact on `.py` count, `.go` count, and `pyproject.toml` / `setup.py` / `setup.cfg` presence.

Validation
- `PYTHONPATH=src python3 -m pytest tests/test_planning.py tests/test_reports.py -q`
- `find . -type f \\( -name '*.py' -o -name '*.go' -o -name 'pyproject.toml' -o -name 'setup.py' -o -name 'setup.cfg' \\) | sed 's#^./##' | awk 'BEGIN{py=0;go=0;pp=0;setup=0} /\\.py$/{py++} /\\.go$/{go++} /pyproject\\.toml$/{pp++} /(setup\\.py|setup\\.cfg)$/{setup++} END{printf("py=%d\\ngo=%d\\npyproject=%d\\nsetup=%d\\n",py,go,pp,setup)}'`
- `git diff --stat && git status --short`
