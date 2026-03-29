# BIG-GO-948 Workpad

## Plan

1. Inventory the remaining `tests/**` Python files and map them against existing `bigclaw-go` Go tests to identify the lane-owned files that still lack Go coverage.
2. Inspect the selected Python tests and the corresponding Go packages to choose the smallest scoped migration slice that can be completed end-to-end in this issue.
3. Implement the missing Go tests or, where direct migration is out of scope, document the concrete deletion or follow-up plan in-repo while keeping changes limited to this lane.
4. Remove the migrated Python test assets that now have Go replacements and keep any untouched Python tests outside this lane unchanged.
5. Run targeted validation commands for the touched Go packages, record exact commands and results, then commit and push the branch.

## Acceptance

- Produce an explicit file list for the `BIG-GO-948` lane.
- Land Go test replacements for the selected remaining Python tests, or document a concrete delete/follow-up plan for any files that cannot be removed in this lane.
- Record exact validation commands, results, and residual risks.
- Reduce Python / non-Go test assets in the repository without widening scope beyond this issue.

## Validation

- `go test` for the exact `bigclaw-go` packages touched by this lane.
- Targeted execution of any new or expanded Go tests covering the migrated Python scenarios.
- `git status --short` to verify the scoped file set before commit.
