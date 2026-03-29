# BIG-GO-948 Workpad

## Scope

Nineteenth-wave cleanup for the remaining Python design-system test by moving its component inventory, console top-bar, information architecture, and UI-acceptance assertions into Go-native parity coverage.

Planned delete set for this continuation:
- `tests/test_design_system.py`

Go coverage used for replacement:
- new `bigclaw-go/internal/designsystemparity/designsystemparity_test.go`

## Acceptance

- Replace the Python design-system test with Go-native parity coverage.
- Keep the changes narrow: port only the contract already exercised by `tests/test_design_system.py` into a dedicated Go package.
- Delete `tests/test_design_system.py` only after the Go replacement exists.
- Update `reports/BIG-GO-948-validation.md` with the new completed file, replacement coverage, command, result, and remaining plan.
- Run targeted Go validation for `bigclaw-go/internal/designsystemparity`.
- Commit and push the continuation changes.

## Validation

- `cd bigclaw-go && go test ./internal/designsystemparity`
- `git status --short`

## Risks

- This slice should stay inside design-system parity only and avoid pulling operations or review-pack ownership into the package.
- The remaining Python operations, reports, and review-pack suites are still intentionally out of scope for this continuation because they need separate bounded replacements.
