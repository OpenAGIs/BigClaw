# BIG-GO-1504 Workpad

## Plan
- Inspect `origin/main` repository state for Python wrapper files in `scripts/` and `scripts/ops/`.
- Confirm whether any compatibility-only Python wrappers still exist elsewhere in scope for this issue.
- If wrappers exist, delete or replace them with non-Python launchers and validate call paths.
- If wrappers do not exist, record the repository-reality blocker with exact evidence, keep changes scoped, and close out with validation.

## Acceptance
- Reduce the actual number of `.py` files in the repository, specifically targeting compatibility-only wrappers in `scripts/` and `scripts/ops/`, if any exist in the checked-out repository state.
- Record before/after counts and the deleted file list against the real checkout.
- Run targeted validation commands tied to the files and commands actually present.
- Commit and push the resulting branch state.

## Validation
- `rg --files | rg '^(scripts|scripts/ops)/.*\\.py$'`
- `rg --files | rg '\\.py$'`
- `find scripts -maxdepth 3 -type f | sort`
- `sed -n '1,40p' scripts/ops/bigclawctl`
- `sed -n '1,40p' scripts/ops/bigclaw-symphony`
- `sed -n '1,40p' scripts/ops/bigclaw-issue`
- `sed -n '1,40p' scripts/ops/bigclaw-panel`
- `git status --short`
