# BIG-GO-923 Workpad

## Plan

1. Inventory the current pytest harness surface, starting with `tests/conftest.py`, test import patterns, and any existing Go-side shared test bootstrap utilities.
2. Implement a scoped Go replacement for the current Python harness responsibility: shared repo-root/bootstrap helpers that let Go tests locate legacy assets and assert migration parity without pytest path injection.
3. Add first-batch Go regression coverage around the new harness so future migration work can move Python test modules into Go packages on top of a stable base.
4. Document the Python/non-Go asset inventory, the Go replacement shape, deletion criteria for `tests/conftest.py`, and exact regression commands.
5. Run targeted validation, then commit and push the scoped change set.

## Acceptance

- The current Python/pytest harness asset list is explicit, including what `tests/conftest.py` does today.
- A Go-side harness replacement or migration scaffold is checked in and covered by targeted Go tests.
- The repo includes a migration note describing the first landed slice, remaining migration path, and the conditions required before deleting the legacy Python harness asset.
- Exact validation commands and results are recorded for the touched scope.

## Validation

- `cd bigclaw-go && go test ./internal/testharness ./internal/regression`
- `cd bigclaw-go && go test ./...`
- `git status --short`
