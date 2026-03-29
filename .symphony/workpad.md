# BIG-GO-948 Workpad

## Scope

Fifth-wave cleanup for the remaining Python DSL tests by adding a narrow Go-native workflow definition runner.

Planned delete set for this continuation:
- `tests/test_dsl.py`

Go coverage used for replacement:
- existing `bigclaw-go/internal/workflow/definition_test.go`
- existing `bigclaw-go/internal/workflow/engine_test.go`
- new `bigclaw-go/internal/workflow/definition_runner.go`
- new `bigclaw-go/internal/workflow/definition_runner_test.go`

## Acceptance

- Replace the Python DSL workflow-definition tests with Go-native coverage.
- Keep the new Go runner narrow: path rendering, invalid-step validation, approval handling, and artifact creation only.
- Delete `tests/test_dsl.py` only after Go parity exists.
- Update `reports/BIG-GO-948-validation.md` with the new completed file, replacements, command, result, and remaining plan.
- Run targeted Go validation for `bigclaw-go/internal/workflow`.
- Commit and push the continuation changes.

## Validation

- `cd bigclaw-go && go test ./internal/workflow`
- `git status --short`

## Risks

- There is no existing Go runner for definition execution, so the new surface must stay deliberately small and only cover what the Python test asserted.
- The larger remaining Python script and report suites are still intentionally out of scope because they are not simple contract-parity deletes.
