# BIG-GO-948 Workpad

## Scope

Seventh-wave cleanup for the remaining Python model round-trip test using existing Go-native coverage.

Planned delete set for this continuation:
- `tests/test_models.py`

Go coverage used for replacement:
- `bigclaw-go/internal/risk/assessment_test.go`
- `bigclaw-go/internal/triage/record_test.go`
- `bigclaw-go/internal/workflow/model_test.go`
- `bigclaw-go/internal/billing/statement_test.go`

## Acceptance

- Remove `tests/test_models.py` only after confirming all asserted round-trip model contracts already exist in Go.
- Keep changes scoped to the delete and lane documentation; no new runtime behavior unless a direct parity gap appears.
- Update `reports/BIG-GO-948-validation.md` with the new completed file, replacement coverage, command, result, and remaining plan.
- Run targeted Go validation for the existing replacement packages.
- Commit and push the continuation changes.

## Validation

- `cd bigclaw-go && go test ./internal/risk`
- `cd bigclaw-go && go test ./internal/triage`
- `cd bigclaw-go && go test ./internal/workflow`
- `cd bigclaw-go && go test ./internal/billing`
- `git status --short`

## Risks

- This slice depends on several existing Go packages rather than one replacement file, so the lane report needs to make the mapping explicit.
- The larger remaining Python script and report suites are still intentionally out of scope because they are not simple contract-parity deletes.
