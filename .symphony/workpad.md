# BIG-GO-1533

## Plan
- Initialize the local worktree from `origin/main` and create a local `BIG-GO-1533` branch if it does not already exist remotely.
- Identify all remaining `.py` files under the BigClaw Go scripts directory, capture exact before evidence, and verify there are no other required changes outside this scope.
- Delete the remaining targeted `.py` files from disk, then capture exact after evidence showing the count reaches zero.
- Run targeted validation commands that prove the before/after counts and the deleted file list.
- Commit the scoped change and push the branch to `origin`.

## Acceptance
- Remaining `.py` files in the targeted BigClaw Go scripts directory are physically removed from the repository worktree.
- Before and after evidence includes exact file paths and exact counts.
- Changes are limited to the file deletions needed for this issue plus this workpad.
- A git commit exists on branch `BIG-GO-1533` and is pushed to `origin`.

## Validation
- Use `find` or `rg --files` to record the exact `.py` files under the target scripts directory before deletion.
- Use the same command after deletion to confirm zero matching files remain.
- Use `git status --short` and `git diff --name-status` to confirm only the intended files were removed.
- Record the exact test and validation commands plus their results in the final report.
