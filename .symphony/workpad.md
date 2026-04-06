# BIG-GO-1501

## Plan
- Materialize the repository from `origin/main` in this workspace because the local checkout is currently empty except for `.git`.
- Measure the current `.py` file count under `src/bigclaw` and identify Python files that are obsolete, replaced by Go, or otherwise safe to delete without broad scope expansion.
- Remove the chosen file(s) or replace them with Go if needed, keeping changes tightly scoped to reducing the real Python file count.
- Run targeted validation covering file-count reduction and any directly impacted tests or build checks.
- Commit the scoped change and push a branch for `BIG-GO-1501`.

## Acceptance
- The actual number of `.py` files under `src/bigclaw` is lower after the change than before.
- The final report includes exact before/after counts and the deleted or replaced file list.
- Validation is tied to repository reality with exact commands and outcomes recorded.
- Changes remain scoped to this migration refill issue.
- The branch is committed and pushed to `origin`.

## Validation
- `find src/bigclaw -type f -name '*.py' | wc -l`
- Additional targeted repo checks based on the specific file(s) removed, such as `go test` for the affected Go package or a repository test script if one directly covers the impacted area.
- `git status --short` to confirm only intended files changed.
