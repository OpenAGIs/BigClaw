# BIG-GO-948 Workpad

## Scope

Ninth-wave cleanup for the remaining Python runtime matrix test by adding a narrow Go-native parity package.

Planned delete set for this continuation:
- `tests/test_runtime_matrix.py`

Go coverage used for replacement:
- new `bigclaw-go/internal/runtimeparity` package
- new `bigclaw-go/internal/runtimeparity/runtimeparity_test.go`

## Acceptance

- Replace the Python runtime matrix test with Go-native parity coverage.
- Keep the new package narrow: medium routing, tool policy enforcement, tool invocation, and worker lifecycle auditing only.
- Delete `tests/test_runtime_matrix.py` only after Go parity exists.
- Update `reports/BIG-GO-948-validation.md` with the new completed file, replacement coverage, command, result, and remaining plan.
- Run targeted Go validation for `bigclaw-go/internal/runtimeparity`.
- Commit and push the continuation changes.

## Validation

- `cd bigclaw-go && go test ./internal/runtimeparity`
- `git status --short`

## Risks

- This slice adds another small Go package; it must stay scoped to the Python runtime matrix contract instead of turning into a second general-purpose worker/runtime stack.
- The larger remaining Python script and report suites are still intentionally out of scope because they are not simple contract-parity deletes.
