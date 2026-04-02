# BIG-GO-947 Workpad

## Plan

1. Inventory the Lane7 repo governance/reporting/risk/planning/mapping/memory/operations/observability Python tests and map each file to an existing Go test, a new Go replacement, or an explicit deletion-plan entry.
2. Keep the implementation scoped to the lane by only touching the mapped Python tests, the required Go tests, and the migration/reporting artifacts for this issue.
3. Add any missing Go coverage needed to replace Python assertions that do not already have repo-native `go test` coverage.
4. Delete Python tests that are fully replaced by Go tests and document any residual Python files that still need follow-up migration.
5. Run targeted `go test` commands for the touched Go packages, capture exact commands and results, then commit and push the branch.

## Acceptance

- Produce an explicit Lane7 file inventory for governance/reporting/risk/planning/mapping/memory/operations/observability tests.
- Land Go test replacements for the migrated Python coverage, or record a concrete deletion plan when the Go replacement is intentionally deferred.
- Reduce Python / non-Go assets where replacement coverage is complete.
- Record exact validation commands, results, and residual risks for the lane.
- Commit the scoped change set and push it to the remote branch.

## Validation

- `cd bigclaw-go && go test ./internal/governance ./internal/repo ./internal/reporting ./internal/risk ./internal/intake ./internal/events ./internal/observability ./internal/api`
- Additional narrower `go test` package commands if new tests are added outside the packages above.
- `cd bigclaw-go && go test ./internal/observability ./internal/reporting`
- `git status --short`

## Status

- Lane migration completed.
- `tests/test_reports.py` and `tests/test_observability.py` were deleted after Go parity landed.
- Branch `BIG-GO-947` is pushed and aligned with `origin/BIG-GO-947`.
