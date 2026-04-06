# BIG-GO-1513 Workpad

## Plan
- Verify the checked-out branch state and establish the baseline count of physical `.py` files in the repository and under `bigclaw-go/scripts`.
- Inspect the current script layout to identify any remaining Python helpers that can be deleted without broadening scope.
- If Python helpers exist, remove the smallest valid target set, update any direct references, and record before/after counts plus deleted-file evidence.
- Run targeted validation commands and capture exact command lines and results.
- Commit the scoped issue artifacts and push `BIG-GO-1513` to `origin`.

## Acceptance
- The repository's physical `.py` file count decreases from the starting baseline.
- The change set stays scoped to this issue's Python-helper deletion sweep.
- Validation includes exact commands and results.
- The branch is committed and pushed.

## Validation
- `find . -type f -name '*.py' | wc -l`
- `git ls-files '*.py' | wc -l`
- `find scripts bigclaw-go/scripts -type f`
- Additional targeted checks only if a deletable Python helper is found.
