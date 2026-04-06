# BIG-GO-1547 Workpad

## Plan
1. Materialize the BigClaw repository in this issue workspace from `origin`.
2. Record the before-state `.py` file count and exact file list.
3. Remove any remaining repository `.py` files, keeping changes scoped to that deletion sweep and evidence updates only.
4. Record the after-state `.py` file count and exact removed-file list.
5. Run targeted validation commands, commit the change on branch `BIG-GO-1547`, and push to `origin`.

## Acceptance
- Repository contains fewer `.py` files after the sweep than before.
- The final change set physically removes the relevant `.py` files from the repository.
- Evidence includes exact before/after counts and the exact removed-file list.
- Targeted validation commands and their results are captured.
- Changes remain scoped to this issue.

## Validation
- `find . -type f -name '*.py' | sed 's#^./##' | sort`
- `find . -path './.git' -prune -o -type f -name '*.py' -print | sed 's#^./##' | sort | wc -l`
- `git status --short`
- `git diff --stat`
- `git add -A && git commit -m 'BIG-GO-1547: sweep residual python files'`
- `git push -u origin BIG-GO-1547`
