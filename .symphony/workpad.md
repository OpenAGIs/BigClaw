# BIG-GO-948 Workpad

## Scope

Seventeenth-wave cleanup for the remaining Python planning test by moving its backlog, entry-gate, and four-week execution-plan assertions into Go-native parity coverage.

Planned delete set for this continuation:
- `tests/test_planning.py`

Go coverage used for replacement:
- new `bigclaw-go/internal/planningparity/planningparity_test.go`

## Acceptance

- Replace the Python planning test with Go-native parity coverage.
- Keep the changes narrow: port only the planning contract already exercised by `tests/test_planning.py` into a dedicated Go package.
- Delete `tests/test_planning.py` only after the Go replacement exists.
- Update `reports/BIG-GO-948-validation.md` with the new completed file, replacement coverage, command, result, and remaining plan.
- Run targeted Go validation for `bigclaw-go/internal/planningparity`.
- Commit and push the continuation changes.

## Validation

- `cd bigclaw-go && go test ./internal/planningparity`
- `git status --short`

## Risks

- This slice should stay inside planning parity only and avoid pulling broader UI, reports, or operations ownership into the package.
- The remaining Python UI and report suites are still intentionally out of scope for this continuation because they need separate bounded replacements.
