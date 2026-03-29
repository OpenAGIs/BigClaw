# BIG-GO-948 Workpad

## Scope

Eighteenth-wave cleanup for the remaining Python console information-architecture test by moving its IA, interaction-draft, and top-bar assertions into Go-native parity coverage.

Planned delete set for this continuation:
- `tests/test_console_ia.py`

Go coverage used for replacement:
- new `bigclaw-go/internal/consoleiaparity/consoleiaparity_test.go`

## Acceptance

- Replace the Python console IA test with Go-native parity coverage.
- Keep the changes narrow: port only the console IA and interaction-draft contract already exercised by `tests/test_console_ia.py` into a dedicated Go package.
- Delete `tests/test_console_ia.py` only after the Go replacement exists.
- Update `reports/BIG-GO-948-validation.md` with the new completed file, replacement coverage, command, result, and remaining plan.
- Run targeted Go validation for `bigclaw-go/internal/consoleiaparity`.
- Commit and push the continuation changes.

## Validation

- `cd bigclaw-go && go test ./internal/consoleiaparity`
- `git status --short`

## Risks

- This slice should stay inside console IA parity only and avoid pulling broader design-system or review-pack ownership into the package.
- The remaining Python UI and report suites are still intentionally out of scope for this continuation because they need separate bounded replacements.
