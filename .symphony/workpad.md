# BIG-GO-948 Workpad

## Scope

Thirteenth-wave cleanup for the remaining Python runtime test by extending the existing Go runtime parity package.

Planned delete set for this continuation:
- `tests/test_runtime.py`

Go coverage used for replacement:
- updated `bigclaw-go/internal/runtimeparity/runtimeparity_test.go`

## Acceptance

- Replace the Python runtime test with Go-native parity coverage.
- Keep the changes narrow: sandbox router profile mapping and paused budget behavior only, layered on the existing runtime parity package.
- Delete `tests/test_runtime.py` only after Go parity exists.
- Update `reports/BIG-GO-948-validation.md` with the new completed file, replacement coverage, command, result, and remaining plan.
- Run targeted Go validation for `bigclaw-go/internal/runtimeparity`.
- Commit and push the continuation changes.

## Validation

- `cd bigclaw-go && go test ./internal/runtimeparity`
- `git status --short`

## Risks

- This slice should stay inside the existing runtime parity package and avoid turning into a second scheduler or worker implementation.
- The larger remaining Python script and report suites are still intentionally out of scope because they are not simple contract-parity deletes.
