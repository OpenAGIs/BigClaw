# BIG-GO-923 Workpad

## Plan

1. Audit the current pytest bootstrap surface in `tests/conftest.py` and the remaining `tests/test_*.py` import patterns, then record the Python/non-Go inventory in the checked-in migration report.
2. Extend `bigclaw-go/internal/testharness` so the Go harness covers the legacy bootstrap responsibilities directly: project/src path resolution, `PYTHONPATH` bootstrapping, and inventory helpers for the remaining pytest surface.
3. Migrate nearby Go tests that still hand-roll cwd/path setup onto the shared harness helpers so the replacement is exercised immediately.
4. Update the migration report with the landed Go replacement, remaining Python-owned runtime slices, deletion conditions for `tests/conftest.py`, and the exact regression commands used here.
5. Run targeted Go validation for the touched packages, capture exact commands/results, then commit and push to the issue branch.

## Acceptance

- The repository explicitly lists the current pytest harness assets and what `tests/conftest.py` still does.
- `bigclaw-go/internal/testharness` contains the Go-native replacement helpers for repo/project/src bootstrap and these helpers are covered by Go tests.
- At least one adjacent Go test slice adopts the shared harness instead of bespoke cwd/path bootstrap.
- The migration report states when `tests/conftest.py` can be deleted and which regression commands gate that removal.
- The final result includes the exact validation commands executed and whether they passed.

## Validation

- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/testharness ./internal/refill ./cmd/bigclawctl`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923 && git status --short`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923 && git add . && git commit -m "..." && git push origin BIG-GO-923-go-test-harness`
