# BIG-GO-1018

## Plan
- Migrate the next scoped residual `tests/**` tranche with existing Go ownership in the file-backed queue.
- Port `tests/test_queue.py` into `bigclaw-go/internal/queue` by adding legacy list-storage compatibility and direct Go tests.
- Remove the migrated Python test file from `tests/`.
- Run targeted Go tests for `bigclaw-go/internal/queue`, capture exact commands and results, then commit and push the branch.

## Acceptance
- Changes stay scoped to this issue's residual `tests/**` tranche.
- The selected Python test behaviors are covered by Go tests against repository code, not tracker metadata.
- The number of repository `.py` files decreases.
- Final report includes impact on `py files`, `go files`, `pyproject.toml`, and `setup.py`.

## Validation
- `go test ./internal/queue`
- `find . -name '*.py' | wc -l`
- `find . -name '*.go' | wc -l`
- `git status --short`
