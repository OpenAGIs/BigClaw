# BIG-GO-1018

## Plan
- Migrate a scoped tranche of residual `tests/**` Python coverage that already maps to existing Go packages.
- Extend Go tests in `bigclaw-go/internal/product` and `bigclaw-go/internal/bootstrap` to absorb the selected Python assertions.
- Remove the migrated Python test files from `tests/`.
- Run targeted Go tests for the touched packages, capture exact commands and results, then commit and push the branch.

## Acceptance
- Changes stay scoped to this issue's residual `tests/**` tranche.
- The selected Python test behaviors are covered by Go tests against repository code, not tracker metadata.
- The number of repository `.py` files decreases.
- Final report includes impact on `py files`, `go files`, `pyproject.toml`, and `setup.py`.

## Validation
- `go test ./bigclaw-go/internal/product ./bigclaw-go/internal/bootstrap`
- `find . -name '*.py' | wc -l`
- `find . -name '*.go' | wc -l`
- `git status --short`
