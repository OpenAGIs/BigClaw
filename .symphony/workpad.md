# BIG-GO-1473

## Plan
- Materialize the repository contents from `origin` into this workspace so the issue can be implemented against the real tree.
- Inventory remaining physical Python files, with emphasis on `scripts` entrypoints/helpers and all repo callers that still reference them.
- Replace eligible callers with existing or new Go command entrypoints, or delete Python assets when they are unused and have clear replacement/delete conditions.
- Remove the targeted Python files and update any related docs or automation references needed to keep the repo functional.
- Run targeted validation proving the Python file count moved down and the Go-based flow works.
- Commit the scoped change set and push the issue branch to `origin`.

## Acceptance
- Remaining Python assets targeted by this issue are physically removed from the repository, not just de-referenced.
- Each removed/migrated Python file has documented replacement Go ownership or an explicit delete condition.
- Validation shows the repository moved closer to Go-only, including an updated Python file count and targeted command/test results.
- Changes stay scoped to `BIG-GO-1473`.

## Validation
- `git fetch origin`
- `git checkout -B BIG-GO-1473 origin/main` or the relevant upstream base branch
- `rg --files -g '*.py' | wc -l`
- Targeted `rg` searches confirming callers no longer reference removed Python entrypoints
- Targeted Go test/build commands for the touched command paths
- `git status --short`

## Findings
- `origin/main` at checkout time already has zero tracked `.py` files, including under `scripts` and `bigclaw-go/scripts`.
- There is no `origin/BIG-GO-1473` branch carrying additional Python assets to delete.
- Because the physical Python file count is already `0`, this issue cannot produce a truthful in-branch file-count reduction; the branch can only document the blocker, lock the zero-Python baseline, and verify Go ownership for the retired paths.
