# BIG-GO-1018

## Plan
- Migrate the next scoped residual `tests/**` tranche for the task memory store.
- Port the small Python memory-pattern store into a dedicated Go package so rule suggestion coverage lives on the Go mainline.
- Remove `tests/test_memory.py` after validating the new Go package tests.
- Remove the migrated Python test file from `tests/`.
- Run targeted Go tests for `bigclaw-go/internal/memory`, capture exact commands and results, then commit and push the branch.

## Acceptance
- Changes stay scoped to this issue's residual `tests/**` tranche.
- The selected Python test behaviors are covered by Go tests against repository code, not tracker metadata.
- The number of repository `.py` files decreases.
- Final report includes impact on `py files`, `go files`, `pyproject.toml`, and `setup.py`.

## Validation
- `go test ./internal/memory`
- `find . -name '*.py' | wc -l`
- `find . -name '*.go' | wc -l`
- `git status --short`

## Results
- `cd bigclaw-go && go test ./internal/memory` -> `ok  	bigclaw-go/internal/memory	1.805s`
- `find . -name '*.py' | wc -l` -> `80`
- `find . -name '*.go' | wc -l` -> `271`
- `git status --short` -> `.symphony/workpad.md` modified; `bigclaw-go/internal/memory/store.go` and `bigclaw-go/internal/memory/store_test.go` added; `tests/test_memory.py` deleted
- Previous completed tranche: `cd bigclaw-go && go test ./internal/repo` -> `ok  	bigclaw-go/internal/repo	0.827s`
- Previous completed tranche: `find . -name '*.py' | wc -l` -> `81`
- Previous completed tranche: `find . -name '*.go' | wc -l` -> `269`
- Previous completed tranche status: `.symphony/workpad.md`, `bigclaw-go/internal/repo/board.go`, `bigclaw-go/internal/repo/repo_surfaces_test.go` modified; `bigclaw-go/internal/repo/collaboration.go` added; `tests/test_repo_collaboration.py` deleted
- Previous completed tranche: `cd bigclaw-go && go test ./internal/workflow` -> `ok  	bigclaw-go/internal/workflow	1.580s`
- Previous completed tranche: `find . -name '*.py' | wc -l` -> `82`
- Previous completed tranche: `find . -name '*.go' | wc -l` -> `268`
- Previous completed tranche status: `.symphony/workpad.md`, `bigclaw-go/internal/workflow/definition.go`, `bigclaw-go/internal/workflow/definition_test.go` modified; `tests/test_dsl.py` deleted
