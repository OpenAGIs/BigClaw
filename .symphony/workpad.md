# BIG-GO-948 Workpad

## Scope

Fourteenth-wave cleanup for the remaining Python evaluation test by adding a narrow Go-native parity package.

Planned delete set for this continuation:
- `tests/test_evaluation.py`

Go coverage used for replacement:
- new `bigclaw-go/internal/evaluationparity` package
- new `bigclaw-go/internal/evaluationparity/evaluationparity_test.go`

## Acceptance

- Replace the Python evaluation test with Go-native parity coverage.
- Keep the new package narrow: benchmark scoring, replay mismatch detection, suite comparison, and small report/detail page rendering only.
- Delete `tests/test_evaluation.py` only after Go parity exists.
- Update `reports/BIG-GO-948-validation.md` with the new completed file, replacement coverage, command, result, and remaining plan.
- Run targeted Go validation for `bigclaw-go/internal/evaluationparity`.
- Commit and push the continuation changes.

## Validation

- `cd bigclaw-go && go test ./internal/evaluationparity`
- `git status --short`

## Risks

- This slice adds another small Go package; it must stay scoped to the Python evaluation contract instead of growing into a general benchmarking framework.
- The larger remaining Python script and report suites are still intentionally out of scope because they are not simple contract-parity deletes.
