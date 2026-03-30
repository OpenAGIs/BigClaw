# BIG-GO-1018

## Plan
- Migrate the next scoped residual `tests/**` tranche for the obsolete live-shadow export path.
- Replace the stale `tests/test_live_shadow_bundle.py` coverage with a Go test for the current `bigclawctl automation migration export-live-shadow-bundle` root-resolution behavior and rely on the existing Go bundle-generation/regression coverage.
- Remove the migrated Python test file from `tests/`.
- Run targeted Go tests for `bigclaw-go/cmd/bigclawctl` and `bigclaw-go/internal/regression`, capture exact commands and results, then commit and push the branch.

## Acceptance
- Changes stay scoped to this issue's residual `tests/**` tranche.
- The selected Python test behaviors are covered by Go tests against repository code, not tracker metadata.
- The number of repository `.py` files decreases.
- Final report includes impact on `py files`, `go files`, `pyproject.toml`, and `setup.py`.

## Validation
- `go test ./cmd/bigclawctl`
- `go test ./internal/regression -run 'TestLiveShadow(ScorecardBundleStaysAligned|BundleSummaryAndIndexStayAligned)'`
- `find . -name '*.py' | wc -l`
- `find . -name '*.go' | wc -l`
- `git status --short`
