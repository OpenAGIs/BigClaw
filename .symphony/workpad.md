# BIG-GO-948 Workpad

## Status

- Completed on branch `big-go-948-lane8-remaining-python-tests`.
- Final commits:
  - `272316b` `BIG-GO-948 remove remaining operations python test`
  - `77862ca` `BIG-GO-948 migrate report parity slice to go`
  - `d1eec65` `BIG-GO-948 remove remaining reports python test`
  - `506dc5b` `BIG-GO-948 remove final ui review python tests`
- Final push: `git push origin big-go-948-lane8-remaining-python-tests` -> success.

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

## Result

- All Python files under `tests/` for this lane were removed.
- Active planning metadata now points at Go-owned validation targets.
- Branch is committed, pushed, and clean after validation.

## Risks

- `bigclaw-go/internal/reviewparity/reviewparity_test.go` is a Go-owned regression shell around the existing Python implementation in `src/bigclaw/ui_review.py`, so the test asset migrated to Go but the production Python module still remains.
- Deleting Python tests requires checked-in planning metadata to stay aligned with the new Go ownership so future validation commands do not drift.
