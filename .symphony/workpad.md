# BIG-GO-948 Workpad

## Scope

Fourth-wave cleanup for a remaining Python-only contract test that can be replaced by a small Go-native package.

Planned delete set for this continuation:
- `tests/test_validation_policy.py`

Go coverage used for replacement:
- new `bigclaw-go/internal/validationpolicy` package
- new `bigclaw-go/internal/validationpolicy/validation_policy_test.go`

## Acceptance

- Replace the Python validation policy logic with a Go-native equivalent and matching tests.
- Delete `tests/test_validation_policy.py` only after Go parity exists.
- Update `reports/BIG-GO-948-validation.md` with the new completed file, Go replacement, command, result, and remaining plan.
- Run targeted Go validation for the new package.
- Commit and push the continuation changes.

## Validation

- `cd bigclaw-go && go test ./internal/validationpolicy`
- `git status --short`

## Risks

- This slice adds a new Go package rather than reusing an existing one, so package naming and scope need to stay narrow and contract-focused.
- The larger remaining Python script and report suites are still intentionally out of scope because they are not simple contract-parity deletes.
