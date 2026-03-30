# BIG-GO-1018

## Plan
- Migrate the next scoped residual `tests/**` tranche for the frozen legacy Python runtime matrix.
- Remove `tests/test_runtime_matrix.py` as stale residual coverage for `src/bigclaw/runtime.py`, which already points to Go mainline replacements, and validate against the current Go worker and scheduler tests.
- Remove the migrated Python test file from `tests/`.
- Run targeted Go tests for `bigclaw-go/internal/worker` and `bigclaw-go/internal/scheduler`, capture exact commands and results, then commit and push the branch.

## Acceptance
- Changes stay scoped to this issue's residual `tests/**` tranche.
- The selected Python test behaviors are covered by Go tests against repository code, not tracker metadata.
- The number of repository `.py` files decreases.
- Final report includes impact on `py files`, `go files`, `pyproject.toml`, and `setup.py`.

## Validation
- `go test ./internal/worker`
- `go test ./internal/scheduler`
- `find . -name '*.py' | wc -l`
- `find . -name '*.go' | wc -l`
- `git status --short`
