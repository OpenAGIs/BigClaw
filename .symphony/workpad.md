# BIG-GO-1030

## Plan
- Confirm the remaining repository-level Python assets and identify whether the final `src/bigclaw/__init__.py` file is still referenced by live repo entrypoints.
- Remove the final physical Python package asset if it is no longer required, and update nearby repo documentation or validation surfaces that still describe `src/bigclaw` as a present implementation tree.
- Run targeted validation for the touched Go legacy-shim surface and record exact commands and results, then commit and push the branch.

## Acceptance
- Repository physical-layer `.py` file count decreases.
- Changes stay scoped to the last residual Python asset and directly related repo guidance.
- Report the resulting `.py` and `.go` file counts plus any `pyproject.toml`/`setup.py`/`setup.cfg` impact.
- Do not rely on tracker-only closure; the repo contents must reflect the reduction.

## Validation
- `find . -type f -name '*.py' | sed 's#^./##' | sort`
- `find . -type f -name '*.go' | sed 's#^./##' | sort | wc -l`
- `find . -type f \\( -name 'pyproject.toml' -o -name 'setup.py' -o -name 'setup.cfg' \\) | sed 's#^./##' | sort`
- `cd bigclaw-go && go test ./internal/legacyshim ./internal/regression`
- `rg -n 'src/bigclaw|python -m bigclaw' README.md workflow.md docs bigclaw-go/docs`
