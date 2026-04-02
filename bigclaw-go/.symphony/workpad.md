# BIG-GO-982 Workpad

## Plan
- Inventory Python files under `scripts/*.py` and `scripts/ops/*.py`, plus the repository-wide Python file count for before/after comparison.
- Inspect the current Go CLI and shell wrapper surfaces that replaced or superseded root/ops script entrypoints.
- Make the smallest scoped changes needed to document the final sweep state for this batch and keep references aligned.
- Run targeted validation for any changed docs/tests, then commit and push the branch.

## Acceptance
- Produce the explicit file list for this batch under `scripts/*.py` and `scripts/ops/*.py`.
- Reduce Python file count in those directories when files still exist there, or document that the batch is already at zero.
- Record keep/replace/delete rationale for the batch scope.
- Report the impact on the repository-wide Python file count.

## Validation
- `find scripts -maxdepth 2 -name '*.py' | sort`
- `find . -name '*.py' | sed 's#^./##' | sort | wc -l`
- Targeted `go test` for any package touched by code changes.
- `git status --short`
