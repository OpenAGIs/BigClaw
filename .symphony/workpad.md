Issue: BIG-GO-1024

Plan
- Move `src/bigclaw/ui_review.py` behind the existing asset-backed compatibility loader in `src/bigclaw/__init__.py`.
- Preserve `bigclaw.ui_review` imports and package-level re-exports while deleting the physical `.py` file from `src/bigclaw`.
- Run targeted UI-review validation plus repository file-count checks, then commit and push the branch.

Acceptance
- `src/bigclaw/ui_review.py` is removed, reducing physical Python file count in `src/bigclaw`.
- `bigclaw.ui_review` remains importable and preserves the tested review-pack, board/report rendering, and bundle export behavior.
- No `pyproject.toml`, `setup.py`, or `setup.cfg` changes are introduced in this tranche.

Validation
- `find src/bigclaw -maxdepth 1 -name '*.py' | sort | wc -l`
- `find . -name '*.py' | wc -l`
- `find . -name '*.go' | wc -l`
- `python3 -m py_compile src/bigclaw/__init__.py tests/test_ui_review.py`
- `PYTHONPATH=src python3 -m pytest tests/test_ui_review.py -q`
- `PYTHONPATH=src python3 - <<'PY'`
- `import bigclaw.ui_review`
- `from bigclaw.ui_review import UIReviewPack`
- `print("module smoke checks passed", bool(UIReviewPack))`
- `PY`
- `git status --short`
- `git diff --stat`
