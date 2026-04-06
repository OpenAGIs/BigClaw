# BIG-GO-1484 Workpad

## Plan
- Verify the current repository state and enumerate any physical Python files under `scripts` and `scripts/ops`.
- If Python wrappers exist in scope, replace them with non-Python equivalents and update any affected invocations.
- Capture exact before/after evidence for repository-wide `.py` file count and in-scope wrapper count.
- Run targeted validation for the touched scripts and record exact commands and results.
- Commit scoped changes and push the branch.

## Acceptance
- No physical Python wrapper files remain under `scripts` or `scripts/ops`.
- Actual repository `.py` file count is reduced by this change.
- Exact before/after evidence is recorded from repository commands, not tracker metadata.
- Targeted validation commands complete successfully.
- Changes remain scoped to this issue.

## Validation
- `git ls-files '*.py' | wc -l`
- `rg --files scripts scripts/ops -g '*.py' | wc -l`
- `find scripts scripts/ops -maxdepth 3 -type f | sort`
- Any targeted script smoke tests needed for touched files

## Current state
- On `origin/main` at `a63c8ec0f999d976a1af890c920a54ac2d6c693a`, `git ls-files '*.py' | wc -l` returns `0`.
- On the same snapshot, `rg --files scripts scripts/ops -g '*.py' | wc -l` returns `0`.
- Current `scripts` wrappers are already shell-based (`scripts/dev_bootstrap.sh`, `scripts/ops/bigclawctl`, `scripts/ops/bigclaw-issue`, `scripts/ops/bigclaw-panel`, `scripts/ops/bigclaw-symphony`).
- Because the repo already contains zero tracked Python files, the required Python-count reduction is currently blocked on upstream state: there is nothing left in scope to remove.

## Validation results
- `git ls-files '*.py' | wc -l` -> `0`
- `rg --files scripts scripts/ops -g '*.py' | wc -l` -> `0`
- `find scripts scripts/ops -maxdepth 3 -type f | sort` ->
  `scripts/dev_bootstrap.sh`
  `scripts/ops/bigclaw-issue`
  `scripts/ops/bigclaw-panel`
  `scripts/ops/bigclaw-symphony`
  `scripts/ops/bigclawctl`
- `bash scripts/ops/bigclaw-issue --help` -> exit `0`
