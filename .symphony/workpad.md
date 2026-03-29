# BIG-GO-948 Workpad

## Plan

1. Confirm the remaining Python test lane scope under `tests/` and map each file to a Go-native owner or deletion path.
2. Delete the remaining Python tests under `tests/` once their contracts are covered by Go-owned validation.
3. Update planning and traceability references so no active lane metadata points at removed Python suites.
4. Refresh `reports/BIG-GO-948-validation.md` with the final lane file list, exact validation commands, results, and residual risks.
5. Run targeted Go tests plus repository status verification, then commit and push the branch.

## Acceptance

- Lane file list is explicit for the remaining Python tests.
- `tests/test_operations.py` is removed only if Go-native replacements already cover its contract.
- `tests/test_reports.py` is removed only if Go-native replacements already cover its contract.
- `tests/test_ui_review.py` is removed only if a Go-owned replacement covers its contract.
- `tests/conftest.py` is removed once no Python tests under `tests/` require it.
- Validation commands and exact results are recorded in `reports/BIG-GO-948-validation.md`.
- Changes remain scoped to `BIG-GO-948`.
- Branch is committed and pushed.

## Validation

- `cd bigclaw-go && go test ./internal/reviewparity ./internal/planningparity ./internal/designsystemparity ./internal/consoleiaparity`
- `cd bigclaw-go && go test ./internal/reporting ./internal/reportingparity`
- `rg --files tests | sort`
- `git status --short`

## Risks

- `bigclaw-go/internal/reviewparity/reviewparity_test.go` is a Go-owned regression shell around the existing Python implementation in `src/bigclaw/ui_review.py`, so the test asset migrated to Go but the production Python module still remains.
- Deleting Python tests requires checked-in planning metadata to stay aligned with the new Go ownership so future validation commands do not drift.
