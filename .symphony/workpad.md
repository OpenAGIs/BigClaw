# BIG-GO-1544

## Plan
1. Capture baseline tracked `.py` counts for the repository and for the in-scope `scripts/*.py` and `scripts/ops/*.py` residual wrapper surface.
2. Physically delete only the remaining in-scope script-wrapper Python files.
3. Recount after deletion and record the exact removed-file list.
4. Run targeted validation commands on the scoped deletion.
5. Commit and push `BIG-GO-1544`.

## Acceptance
- Remaining root `scripts/*.py` files are physically removed.
- Remaining `scripts/ops/*.py` wrapper files are physically removed.
- Before/after counts are recorded for repo-wide tracked `.py` files and the in-scope script-wrapper subset.
- The exact removed-file list is recorded.
- Validation commands and exact results are recorded.
- Changes remain scoped to this issue.

## Validation
- `git ls-tree -r --name-only HEAD | rg '\\.py$' | wc -l`
- `git ls-tree -r --name-only HEAD | rg '^scripts(/ops)?/.*\\.py$' | wc -l`
- `git ls-tree -r --name-only HEAD | rg '^scripts(/ops)?/.*\\.py$' | sort`
- `git diff --name-only --diff-filter=D | sort`
- `git status --short`

## Baseline
- Repo-wide tracked `.py` files before deletion: `115`
- In-scope tracked script-wrapper `.py` files before deletion: `7`
- In-scope files before deletion:
  - `scripts/create_issues.py`
  - `scripts/dev_smoke.py`
  - `scripts/ops/bigclaw_github_sync.py`
  - `scripts/ops/bigclaw_refill_queue.py`
  - `scripts/ops/bigclaw_workspace_bootstrap.py`
  - `scripts/ops/symphony_workspace_bootstrap.py`
  - `scripts/ops/symphony_workspace_validate.py`

## Validation Results
- `git ls-tree -r --name-only HEAD | rg '\\.py$' | wc -l` -> `115` before deletion
- `git ls-tree -r --name-only HEAD | rg '^scripts(/ops)?/.*\\.py$' | wc -l` -> `7` before deletion
- `find . -type f -name '*.py' | sed 's#^./##' | sort | wc -l` -> `108` after deletion in the working tree
- `find scripts -type f -name '*.py' | sed 's#^./##' | sort | wc -l` -> `0` after deletion in the working tree
- `find scripts/ops -maxdepth 1 -type f | sed 's#^./##' | sort` -> `scripts/ops/bigclaw-issue`, `scripts/ops/bigclaw-panel`, `scripts/ops/bigclaw-symphony`, `scripts/ops/bigclawctl`
- `git diff --name-only --diff-filter=D | sort` -> the seven deleted files listed below
- `git diff --check` -> no output

## Exact Removed-File List
- `scripts/create_issues.py`
- `scripts/dev_smoke.py`
- `scripts/ops/bigclaw_github_sync.py`
- `scripts/ops/bigclaw_refill_queue.py`
- `scripts/ops/bigclaw_workspace_bootstrap.py`
- `scripts/ops/symphony_workspace_bootstrap.py`
- `scripts/ops/symphony_workspace_validate.py`

## Outcome
- Repo-wide `.py` filesystem count moved from `115` to `108`.
- In-scope script-wrapper `.py` filesystem count moved from `7` to `0`.
- The remaining `scripts/ops` wrappers are non-Python executables only.
