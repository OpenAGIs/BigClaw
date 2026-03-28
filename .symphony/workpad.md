# BIG-GO-924 Workpad

## Plan

1. Inspect the Python governance assets and the current Go replacements under `bigclaw-go/internal` to identify exact parity gaps for `tests/test_repo_governance.py`, `tests/test_governance.py`, and `tests/test_repo_board.py`.
2. Keep the migration scoped to repository governance concerns by landing the missing Go implementation and tests for the repo board surface, plus any small documentation needed to state migration status and Python removal conditions.
3. Record the current Python and non-Go governance asset inventory together with the Go replacement mapping and the regression commands required before deleting the old Python assets.
4. Run targeted Go tests for the touched governance packages, capture exact commands and results, then commit and push the branch.

## Acceptance

- Document the current Python or non-Go governance assets that still back the targeted tests.
- Produce a concrete Go migration mapping and deletion criteria for the legacy Python governance assets.
- Land first-batch Go implementation and tests that cover the missing governance migration surface without broadening scope.
- Record exact validation commands and outcomes for the touched governance packages.

## Validation

- `go test ./internal/repo ./internal/governance`
- If implementation changes stay confined to repo governance packages, do not expand beyond targeted package tests.
- Capture the exact command output status in the final report after the run.

