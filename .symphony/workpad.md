# BIG-GO-948 Workpad

## Scope

Eleventh-wave cleanup for the remaining Python control center test by adding a narrow Go-native parity package.

Planned delete set for this continuation:
- `tests/test_control_center.py`

Go coverage used for replacement:
- new `bigclaw-go/internal/controlcenterparity` package
- new `bigclaw-go/internal/controlcenterparity/controlcenterparity_test.go`

## Acceptance

- Replace the Python control center test with Go-native parity coverage.
- Keep the new package narrow: persistent queue ordering, queue control center aggregation, actions, and shared-view empty-state rendering only.
- Delete `tests/test_control_center.py` only after Go parity exists.
- Update `reports/BIG-GO-948-validation.md` with the new completed file, replacement coverage, command, result, and remaining plan.
- Run targeted Go validation for `bigclaw-go/internal/controlcenterparity`.
- Commit and push the continuation changes.

## Validation

- `cd bigclaw-go && go test ./internal/controlcenterparity`
- `git status --short`

## Risks

- This slice adds another small Go package; it must stay scoped to the Python control center contract instead of changing the established Go reporting surface.
- The larger remaining Python script and report suites are still intentionally out of scope because they are not simple contract-parity deletes.
