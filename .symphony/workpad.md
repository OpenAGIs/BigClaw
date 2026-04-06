# BIG-GO-1523

## Plan

1. Capture the current repository `.py` count and enumerate the Python files under `bigclaw-go/scripts`.
2. Physically delete only the remaining `bigclaw-go/scripts/*.py` files from disk.
3. Run targeted validation to confirm the files are gone and the repository `.py` count decreased.
4. Commit the scoped deletion-only change and push `BIG-GO-1523` to the remote.

## Acceptance

- The repository `.py` count is lower after the change than before.
- Only `bigclaw-go/scripts` Python files are physically deleted for this issue.
- Exact removed-file evidence is captured from git status / diff output.
- Validation commands and results are recorded.
- The branch is committed and pushed.

## Validation

- `rg --files -g '*.py' | wc -l`
- `rg --files bigclaw-go/scripts -g '*.py'`
- `git diff --name-status --diff-filter=D`
- `git status --short`
