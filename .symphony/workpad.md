# BIG-GO-1486

## Plan
1. Initialize this worktree against the configured `origin` remote and check out the target base branch.
2. Inventory repository Python files, focusing on workspace/bootstrap/planning helper paths still counted in repo inventory.
3. Remove or replace the remaining in-scope Python helper files with non-Python equivalents while keeping behavior intact.
4. Re-run inventory and targeted validation to capture exact before/after evidence.
5. Commit the scoped change set and push the branch to `origin`.

## Acceptance
- Repository Python file count is reduced by this change.
- Remaining edits are scoped to the workspace/bootstrap/planning helper cleanup for `BIG-GO-1486`.
- Exact before/after evidence for Python file inventory is captured from commands run in this worktree.
- Targeted validation commands are executed and their results recorded.
- Changes are committed and pushed to the remote branch for this issue.

## Validation
- `git fetch origin --prune`
- `git checkout -B BIG-GO-1486 origin/main` or the appropriate default branch
- `rg --files -g '*.py' .`
- Targeted repo checks for the affected bootstrap/planning paths after replacement
- `git status --short`
- `git log -1 --stat`

## Execution Notes
- `origin/main` materialized at baseline `a63c8ec`.
- Repository-wide physical `.py` inventory was already `0` before edits in this workspace.
- Lane implementation therefore adds workspace/bootstrap/planning regression coverage plus exact zero-inventory evidence instead of removing in-branch Python files.
- Targeted validation passed:
  - `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort` -> no output
  - `find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort` -> no output
  - `cd bigclaw-go && go test -count=1 ./internal/bootstrap ./internal/planning ./internal/regression -run 'TestBIGGO1486(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|WorkspaceBootstrapAndPlanningPathsRemainAvailable|LaneReportCapturesSweepState)$'` -> `ok   bigclaw-go/internal/bootstrap 0.163s [no tests to run]`, `ok   bigclaw-go/internal/planning 0.254s [no tests to run]`, `ok   bigclaw-go/internal/regression 0.286s`
