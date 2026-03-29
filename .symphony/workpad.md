# BIG-GO-948 Workpad

## Scope

Fifteenth-wave cleanup for the remaining Python parallel validation bundle test by moving its synthetic exporter assertions into Go regression coverage.

Planned delete set for this continuation:
- `tests/test_parallel_validation_bundle.py`

Go coverage used for replacement:
- updated `bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go`

## Acceptance

- Replace the Python parallel validation bundle test with Go-native regression coverage.
- Keep the changes narrow: exercise the existing exporter script from Go against a synthetic bundle fixture and assert only the contract fields already covered by the Python test.
- Delete `tests/test_parallel_validation_bundle.py` only after Go parity exists.
- Update `reports/BIG-GO-948-validation.md` with the new completed file, replacement coverage, command, result, and remaining plan.
- Run targeted Go validation for `bigclaw-go/internal/regression`.
- Commit and push the continuation changes.

## Validation

- `cd bigclaw-go && go test ./internal/regression -run TestLane8ExportValidationBundleGeneratesLatestReportsAndIndex`
- `git status --short`

## Risks

- This slice should stay inside regression coverage and avoid taking ownership of the exporter implementation itself.
- The larger remaining Python script and report suites are still intentionally out of scope because they are not simple contract-parity deletes.
