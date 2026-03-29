# BIG-GO-948 Workpad

## Scope

Third-wave cleanup for remaining Python tests that already have direct Go-native replacements or only need a small Go test to close the gap.

Planned delete set for this continuation:
- `tests/test_governance.py`
- `tests/test_workspace_bootstrap.py`

Go coverage used for replacement:
- `bigclaw-go/internal/governance/freeze_test.go`
- `bigclaw-go/internal/bootstrap/bootstrap_test.go`
- new Go coverage for `bigclaw-go/internal/bootstrap.BuildValidationReport`

## Acceptance

- Extend `BIG-GO-948` with another scoped Python test reduction wave.
- Add any missing Go test coverage needed before deleting the matching Python file.
- Update `reports/BIG-GO-948-validation.md` with the expanded completed file list, replacement coverage, validation commands, and residual plan.
- Run targeted Go validation for governance and bootstrap coverage.
- Commit and push the continuation changes.

## Validation

- `cd bigclaw-go && go test ./internal/governance`
- `cd bigclaw-go && go test ./internal/bootstrap`
- `git status --short`

## Risks

- `tests/test_workspace_bootstrap.py` includes a validation-report assertion that was not yet covered in Go; deletion is only safe after adding that coverage.
- The remaining Python script and report suites are still intentionally out of scope because they are not simple contract-parity deletes.
