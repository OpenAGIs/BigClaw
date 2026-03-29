# BIG-GO-948 Workpad

## Scope

Sixteenth-wave cleanup for the remaining Python live shadow bundle test by moving its synthetic exporter assertions into Go regression coverage.

Planned delete set for this continuation:
- `tests/test_live_shadow_bundle.py`

Go coverage used for replacement:
- new `bigclaw-go/internal/regression/live_shadow_bundle_export_test.go`

## Acceptance

- Replace the Python live shadow bundle test with Go-native regression coverage.
- Keep the changes narrow: exercise the existing migration exporter script from Go against synthetic shadow fixtures and assert only the bundle summary/index/rollup contract already covered by the Python test.
- Delete `tests/test_live_shadow_bundle.py` only after Go parity exists.
- Update `reports/BIG-GO-948-validation.md` with the new completed file, replacement coverage, command, result, and remaining plan.
- Run targeted Go validation for `bigclaw-go/internal/regression`.
- Commit and push the continuation changes.

## Validation

- `cd bigclaw-go && go test ./internal/regression -run TestLane8ExportLiveShadowBundle`
- `git status --short`

## Risks

- This slice should stay inside regression coverage and avoid taking ownership of the migration exporter implementation itself.
- The larger remaining Python script and report suites are still intentionally out of scope because they are not simple contract-parity deletes.
