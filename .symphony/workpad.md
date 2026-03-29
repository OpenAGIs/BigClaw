# BIG-GO-948 Workpad

## Scope

Eighth-wave cleanup for the remaining Python event bus test by adding a narrow Go-native parity package.

Planned delete set for this continuation:
- `tests/test_event_bus.py`

Go coverage used for replacement:
- new `bigclaw-go/internal/runbus` package
- new `bigclaw-go/internal/runbus/runbus_test.go`

## Acceptance

- Replace the Python event bus test with Go-native parity coverage.
- Keep the new package narrow: run registration/loading, event recording, status transitions, comment emission, subscribers, and ledger persistence only.
- Delete `tests/test_event_bus.py` only after Go parity exists.
- Update `reports/BIG-GO-948-validation.md` with the new completed file, replacement coverage, command, result, and remaining plan.
- Run targeted Go validation for `bigclaw-go/internal/runbus`.
- Commit and push the continuation changes.

## Validation

- `cd bigclaw-go && go test ./internal/runbus`
- `git status --short`

## Risks

- This slice adds another small Go package; it must stay scoped to the Python event bus contract instead of turning into a generic workflow system.
- The larger remaining Python script and report suites are still intentionally out of scope because they are not simple contract-parity deletes.
