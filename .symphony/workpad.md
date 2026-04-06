## Plan

1. Establish the actual repository baseline from the current `main` tree and record the real `.py` file count.
2. Inspect the working tree for any physically present Python files that can be removed without broadening scope.
3. If removable Python files exist, delete only those files and record exact before/after evidence.
4. Run targeted validation commands that prove the final repository `.py` count and show any removed-file evidence.
5. Commit the result on `BIG-GO-1530` and push the branch to `origin`.

## Acceptance

- The repository shows a verified before/after `.py` file count.
- Any removed Python files are listed explicitly as exact evidence.
- Validation commands and their results are recorded.
- Changes remain scoped to this issue.
- The branch is committed and pushed.

## Validation

- `rg --files -g '*.py' | wc -l`
- `find . -name '*.py' | sort`
- `git status --short`
- If deletions occur: `git diff --name-status --diff-filter=D`

## Repo Reality

- Current source tree was materialized from `origin/main` commit `646edf33f62c20ccbc4af7c99c27312e1a4c6069`.
- Baseline `.py` count is currently `0`, which is already below the issue target threshold of `130`.
- If no Python files are present, the success condition "decreasing the actual number of .py files" is blocked by repository reality and must be reported as such with exact command evidence.
