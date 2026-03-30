# BIG-GO-1018

## Plan
- Migrate the next scoped residual `tests/**` tranche with existing ownership in the validation bundle export script.
- Port `tests/test_parallel_validation_bundle.py` into `bigclaw-go/scripts/e2e/export_validation_bundle_test.py` by moving the end-to-end bundle export scenario into the existing script-local test file.
- Remove the migrated Python test file from `tests/`.
- Run targeted script tests for `bigclaw-go/scripts/e2e/export_validation_bundle_test.py`, capture exact commands and results, then commit and push the branch.

## Acceptance
- Changes stay scoped to this issue's residual `tests/**` tranche.
- The selected Python test behaviors are covered by Go tests against repository code, not tracker metadata.
- The number of repository `.py` files decreases.
- Final report includes impact on `py files`, `go files`, `pyproject.toml`, and `setup.py`.

## Validation
- `python3 bigclaw-go/scripts/e2e/export_validation_bundle_test.py`
- `find . -name '*.py' | wc -l`
- `find . -name '*.go' | wc -l`
- `git status --short`
