# BIG-GO-1018

## Plan
- Migrate the next scoped residual `tests/**` tranche for the obsolete Python scheduler contract.
- Add direct Go scheduler tests that lock the current routing and reason strings in `bigclaw-go/internal/scheduler`, then remove `tests/test_scheduler.py` as superseded by the Go scheduler semantics.
- Remove the migrated Python test file from `tests/`.
- Run targeted Go tests for `bigclaw-go/internal/scheduler`, capture exact commands and results, then commit and push the branch.

## Acceptance
- Changes stay scoped to this issue's residual `tests/**` tranche.
- The selected Python test behaviors are covered by Go tests against repository code, not tracker metadata.
- The number of repository `.py` files decreases.
- Final report includes impact on `py files`, `go files`, `pyproject.toml`, and `setup.py`.

## Validation
- `go test ./internal/scheduler`
- `find . -name '*.py' | wc -l`
- `find . -name '*.go' | wc -l`
- `git status --short`
