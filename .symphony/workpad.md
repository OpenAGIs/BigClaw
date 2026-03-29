# BIG-GO-948 Workpad

## Scope

Twelfth-wave cleanup for the remaining Python continuation policy gate test by adding a narrow Go-native parity package.

Planned delete set for this continuation:
- `tests/test_validation_bundle_continuation_policy_gate.py`

Go coverage used for replacement:
- new `bigclaw-go/internal/policygateparity` package
- new `bigclaw-go/internal/policygateparity/policygateparity_test.go`

## Acceptance

- Replace the Python continuation policy gate test with Go-native parity coverage.
- Keep the new package narrow: scorecard policy evaluation, checked-in report assertions, and CLI exit-code verification only.
- Delete `tests/test_validation_bundle_continuation_policy_gate.py` only after Go parity exists.
- Update `reports/BIG-GO-948-validation.md` with the new completed file, replacement coverage, command, result, and remaining plan.
- Run targeted Go validation for `bigclaw-go/internal/policygateparity`.
- Commit and push the continuation changes.

## Validation

- `cd bigclaw-go && go test ./internal/policygateparity`
- `git status --short`

## Risks

- This slice adds another small Go package; it must stay scoped to the Python continuation policy gate contract instead of taking ownership of the broader e2e Python script pipeline.
- The larger remaining Python script and report suites are still intentionally out of scope because they are not simple contract-parity deletes.
