Issue: BIG-GO-1030

Plan
- Fold `src/bigclaw/ui_review.py` into `src/bigclaw/reports.py`.
- Preserve compatibility for `bigclaw.ui_review` imports through the package legacy-surface shim in `src/bigclaw/__init__.py`.
- Re-run targeted UI-review pytest coverage plus a direct import-compatibility check, recalculate repository `.py` / `.go` / `pyproject` / `setup` counts, then commit and push.

Acceptance
- The repository physical `.py` file count decreases again.
- `src/bigclaw/ui_review.py` is removed from the tree.
- UI review behaviors still pass from `src/bigclaw/reports.py` and `tests/test_ui_review.py`.
- `from bigclaw.ui_review import ...` continues to resolve through the compatibility surface.
- Final report includes the exact impact on `.py` count, `.go` count, and `pyproject.toml` / `setup.py` / `setup.cfg` presence.

Validation
- `PYTHONPATH=src python3 -m pytest tests/test_ui_review.py -q`
- `PYTHONPATH=src python3 - <<'PY'\nfrom bigclaw.ui_review import UIReviewPackAuditor, render_ui_review_pack_report\nprint(UIReviewPackAuditor.__name__)\nprint(render_ui_review_pack_report.__name__)\nPY`
- `find . -type f \( -name '*.py' -o -name '*.go' -o -name 'pyproject.toml' -o -name 'setup.py' -o -name 'setup.cfg' \) | sed 's#^./##' | awk 'BEGIN{py=0;go=0;pp=0;setup=0} /\.py$/{py++} /\.go$/{go++} /pyproject\.toml$/{pp++} /(setup\.py|setup\.cfg)$/{setup++} END{printf("py=%d\ngo=%d\npyproject=%d\nsetup=%d\n",py,go,pp,setup)}'`
- `git diff --stat && git status --short`
