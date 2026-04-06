# BIG-GO-1487

## Plan
1. Materialize the repository from `origin` because the local checkout currently has no valid `HEAD` and only contains `.git`.
2. Measure Python file counts by directory and select the largest directory for a smallest-valid refill sweep.
3. Delete only Python assets in the chosen directory when they are not required for Go-only migration scope.
4. Record exact before/after Python counts and the deleted file list in committed evidence.
5. Run targeted validation commands, then commit and push the branch.

## Acceptance
- Repository Python file count is reduced in a real commit.
- The sweep is largest-directory-first based on measured counts.
- The change includes exact before/after counts and the deleted file list.
- Validation commands and results are recorded.
- Changes stay scoped to this issue.

## Validation
- `git fetch origin`
- `git checkout -B BIG-GO-1487 origin/main` or the appropriate base branch
- `find . -type f -name '*.py' | wc -l`
- Directory-level Python count command to prove largest-directory-first selection
- `git diff --stat`
- Targeted tests or validation commands relevant to the deleted assets
- `git status --short`

## Results
- Materialized the issue workspace from `origin/BIG-GO-1480`, the nearest recent branch with residual Python assets (`23` total `*.py` files).
- Largest directory before sweep: `bigclaw-go/scripts/e2e` with `15` Python files.
- Total Python files after sweep: `8`.
- Evidence file: `bigclaw-go/docs/reports/big-go-1487-python-asset-sweep.md`.
