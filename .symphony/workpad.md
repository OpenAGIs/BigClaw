# BIG-GO-948 Workpad

## Plan

1. Confirm the remaining Python test lane scope under `tests/` and map each file to existing Go-native coverage or a residual delete plan.
2. Delete Python tests whose contract is already covered in `bigclaw-go`, keeping the change scoped to the lane.
3. Update planning/reporting references that still point at removed Python test files so checked-in traceability stays accurate.
4. Refresh `reports/BIG-GO-948-validation.md` with the lane file list, replacement coverage, exact validation commands, results, and residual risks.
5. Run targeted Go tests plus repository status verification, then commit and push the branch.

## Acceptance

- Lane file list is explicit for the remaining Python tests.
- `tests/test_operations.py` and `tests/test_reports.py` are removed only if Go-native replacements already cover their contracts.
- `tests/test_ui_review.py` is either replaced or left with an explicit delete/migration plan and stated risk.
- Validation commands and exact results are recorded in `reports/BIG-GO-948-validation.md`.
- Changes remain scoped to `BIG-GO-948`.
- Branch is committed and pushed.

## Validation

- `cd bigclaw-go && go test ./internal/reporting`
- `cd bigclaw-go && go test ./internal/planningparity`
- `git status --short`

## Risks

- `tests/test_ui_review.py` currently appears to have no Go-native owner in this branch; if that remains true, this issue can only leave a bounded delete plan rather than removing it outright.
- Deleting Python tests requires updating checked-in planning metadata that still references pytest commands and file targets.
