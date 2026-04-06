# BIG-GO-1482

## Plan
- Inventory the remaining checked-in Python files under `tests` and capture the before count.
- Delete the legacy `tests` Python tree, including `conftest.py` and bootstrap-related test files.
- Update live checked-in callers that still point at the deleted Python tests so the repo stays internally consistent.
- Run targeted validation and record exact before/after evidence plus command results.
- Commit the scoped changes and push branch `BIG-GO-1482`.

## Acceptance
- Tracked `.py` files under `tests` are materially reduced from the branch baseline.
- `tests/conftest.py` and `tests/test_workspace_bootstrap.py` are removed with the rest of the legacy Python test tree.
- Active repo callers no longer require the deleted `tests/*.py` paths.
- Targeted validation commands are recorded with exact results.
- The branch is committed and pushed to `origin/BIG-GO-1482`.

## Validation
- `find tests -maxdepth 1 -type f -name '*.py' | sort | wc -l`
- `find tests -maxdepth 1 -type f -name '*.py' | sort`
- `cd bigclaw-go && go test ./internal/bootstrap ./internal/legacyshim ./cmd/bigclawctl`
- `python3 -m compileall src scripts/create_issues.py scripts/dev_smoke.py bigclaw-go/scripts`
- `git status --short`
- `git log --oneline -1`
