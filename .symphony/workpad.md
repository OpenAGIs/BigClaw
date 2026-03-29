# BIG-GO-948 Workpad

## Scope

Sixth-wave cleanup for the remaining Python memory store test by adding a narrow Go-native parity package.

Planned delete set for this continuation:
- `tests/test_memory.py`

Go coverage used for replacement:
- new `bigclaw-go/internal/memory` package
- new `bigclaw-go/internal/memory/store_test.go`

## Acceptance

- Replace the Python memory store test with Go-native coverage.
- Keep the new package narrow: persisted patterns, success recording, overlap scoring, and merged suggestions only.
- Delete `tests/test_memory.py` only after Go parity exists.
- Update `reports/BIG-GO-948-validation.md` with the new completed file, replacements, command, result, and remaining plan.
- Run targeted Go validation for `bigclaw-go/internal/memory`.
- Commit and push the continuation changes.

## Validation

- `cd bigclaw-go && go test ./internal/memory`
- `git status --short`

## Risks

- This slice adds another small Go package, so it needs to stay minimal and avoid turning into a broader recommendation engine.
- The larger remaining Python script and report suites are still intentionally out of scope because they are not simple contract-parity deletes.
