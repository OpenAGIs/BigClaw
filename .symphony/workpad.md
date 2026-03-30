# BIG-GO-1018

## Plan
- Migrate the next scoped residual `tests/**` tranche for repo collaboration thread merging.
- Add the bounded collaboration thread helpers to the Go repo package so repo discussion board posts can be merged with native decision notes without relying on the legacy Python surface.
- Remove `tests/test_repo_collaboration.py` after validating the repo package tests.
- Remove the migrated Python test file from `tests/`.
- Run targeted Go tests for `bigclaw-go/internal/repo`, capture exact commands and results, then commit and push the branch.

## Acceptance
- Changes stay scoped to this issue's residual `tests/**` tranche.
- The selected Python test behaviors are covered by Go tests against repository code, not tracker metadata.
- The number of repository `.py` files decreases.
- Final report includes impact on `py files`, `go files`, `pyproject.toml`, and `setup.py`.

## Validation
- `go test ./internal/repo`
- `find . -name '*.py' | wc -l`
- `find . -name '*.go' | wc -l`
- `git status --short`

## Results
- `cd bigclaw-go && go test ./internal/repo` -> `ok  	bigclaw-go/internal/repo	0.827s`
- `find . -name '*.py' | wc -l` -> `81`
- `find . -name '*.go' | wc -l` -> `269`
- `git status --short` -> `.symphony/workpad.md`, `bigclaw-go/internal/repo/board.go`, `bigclaw-go/internal/repo/repo_surfaces_test.go` modified; `bigclaw-go/internal/repo/collaboration.go` added; `tests/test_repo_collaboration.py` deleted
- Previous completed tranche: `cd bigclaw-go && go test ./internal/workflow` -> `ok  	bigclaw-go/internal/workflow	1.580s`
- Previous completed tranche: `find . -name '*.py' | wc -l` -> `82`
- Previous completed tranche: `find . -name '*.go' | wc -l` -> `268`
- Previous completed tranche status: `.symphony/workpad.md`, `bigclaw-go/internal/workflow/definition.go`, `bigclaw-go/internal/workflow/definition_test.go` modified; `tests/test_dsl.py` deleted
