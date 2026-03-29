# BIG-GO-948 Workpad

## Scope

Tenth-wave cleanup for the remaining Python execution flow test by adding a narrow Go-native parity package.

Planned delete set for this continuation:
- `tests/test_execution_flow.py`

Go coverage used for replacement:
- new `bigclaw-go/internal/executionparity` package
- new `bigclaw-go/internal/executionparity/executionparity_test.go`

## Acceptance

- Replace the Python execution flow test with Go-native parity coverage.
- Keep the new package narrow: queue dequeue, scheduler decision recording, report emission, and ledger persistence only.
- Delete `tests/test_execution_flow.py` only after Go parity exists.
- Update `reports/BIG-GO-948-validation.md` with the new completed file, replacement coverage, command, result, and remaining plan.
- Run targeted Go validation for `bigclaw-go/internal/executionparity`.
- Commit and push the continuation changes.

## Validation

- `cd bigclaw-go && go test ./internal/executionparity`
- `git status --short`

## Risks

- This slice adds another small Go package; it must stay scoped to the Python execution flow contract instead of turning into another scheduler/runtime implementation.
- The larger remaining Python script and report suites are still intentionally out of scope because they are not simple contract-parity deletes.
