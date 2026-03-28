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
- The current `tests/conftest.py` deletion blockers are machine-checked from Go rather than only described in prose.
- The current `tests/conftest.py` delete-readiness summary is available as one stable line from Go-owned harness code.
- The final result includes the exact validation commands executed and whether they passed.

## Validation

- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923 && python3 -m pytest tests/test_mapping.py -q`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/testharness ./internal/refill ./internal/legacyshim ./cmd/bigclawctl`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923 && git status --short`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923 && git add . && git commit -m "..." && git push origin BIG-GO-923-go-test-harness`

## Validation Results

- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923 && python3 -m pytest tests/test_mapping.py -q`
  Result: passed (`.. [100%]`; re-run after latest harness and deletion-gate changes)
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/testharness ./internal/refill ./internal/legacyshim ./cmd/bigclawctl`
  Result: passed (`ok` for `internal/testharness`, `internal/refill`, `internal/legacyshim`, `cmd/bigclawctl`; re-run after latest harness and deletion-gate changes; includes Go-side `PYTHONPATH` import smoke for `bigclaw.mapping`, a Go-launched `pytest tests/test_mapping.py -q` smoke via shared harness, checked-in legacy shim `py_compile` coverage from Go, `cmd/bigclawctl` adoption of shared executable probing, and current `conftest` deletion-gate assertions)

## Current Status

- `tests/conftest.py` delete-readiness: `conftest_delete_ready=false blockers=56 legacy pytest modules remain under tests/; 47 legacy pytest modules still import bigclaw from src/; 3 legacy pytest modules still import pytest directly`
