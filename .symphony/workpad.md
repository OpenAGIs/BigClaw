# BIG-GO-1018

## Plan
- Migrate the next scoped residual `tests/**` tranche for the workflow definition DSL surface.
- Mirror the Python `WorkflowDefinition.validate()` contract in Go definition parsing so invalid step kinds are rejected by the Go-owned workflow package.
- Remove `tests/test_dsl.py` after validating definition parsing, closeout rendering, and workflow acceptance behavior in Go tests.
- Remove the migrated Python test file from `tests/`.
- Run targeted Go tests for `bigclaw-go/internal/workflow`, capture exact commands and results, then commit and push the branch.

## Acceptance
- Changes stay scoped to this issue's residual `tests/**` tranche.
- The selected Python test behaviors are covered by Go tests against repository code, not tracker metadata.
- The number of repository `.py` files decreases.
- Final report includes impact on `py files`, `go files`, `pyproject.toml`, and `setup.py`.

## Validation
- `go test ./internal/workflow`
- `find . -name '*.py' | wc -l`
- `find . -name '*.go' | wc -l`
- `git status --short`

## Results
- `cd bigclaw-go && go test ./internal/workflow` -> `ok  	bigclaw-go/internal/workflow	1.580s`
- `find . -name '*.py' | wc -l` -> `82`
- `find . -name '*.go' | wc -l` -> `268`
- `git status --short` -> `.symphony/workpad.md`, `bigclaw-go/internal/workflow/definition.go`, `bigclaw-go/internal/workflow/definition_test.go` modified; `tests/test_dsl.py` deleted
