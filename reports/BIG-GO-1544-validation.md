# BIG-GO-1544 Validation

## Scope

Physically remove the remaining root `scripts/*.py` files and `scripts/ops/*.py` wrapper files, and record exact before/after evidence plus the exact removed-file list.

## Before/After Counts

- Repo-wide `.py` files: `115 -> 108`
- In-scope script-wrapper `.py` files: `7 -> 0`

## Exact Removed-File List

- `scripts/create_issues.py`
- `scripts/dev_smoke.py`
- `scripts/ops/bigclaw_github_sync.py`
- `scripts/ops/bigclaw_refill_queue.py`
- `scripts/ops/bigclaw_workspace_bootstrap.py`
- `scripts/ops/symphony_workspace_bootstrap.py`
- `scripts/ops/symphony_workspace_validate.py`

## Validation Commands

- `git ls-tree -r --name-only HEAD | rg '\\.py$' | wc -l` -> `115`
- `git ls-tree -r --name-only HEAD | rg '^scripts(/ops)?/.*\\.py$' | wc -l` -> `7`
- `git ls-tree -r --name-only HEAD | rg '^scripts(/ops)?/.*\\.py$' | sort` -> seven in-scope files listed above
- `find . -type f -name '*.py' | sed 's#^./##' | sort | wc -l` -> `108`
- `find scripts -type f -name '*.py' | sed 's#^./##' | sort | wc -l` -> `0`
- `find scripts/ops -maxdepth 1 -type f | sed 's#^./##' | sort` -> `scripts/ops/bigclaw-issue`, `scripts/ops/bigclaw-panel`, `scripts/ops/bigclaw-symphony`, `scripts/ops/bigclawctl`
- `git diff --name-only --diff-filter=D | sort` -> seven deleted files listed above
- `git diff --check` -> no output

## Notes

The deletion was intentionally scoped to the remaining root `scripts` and `scripts/ops` Python wrappers only. Other Python files still exist elsewhere in the repository and are out of scope for `BIG-GO-1544`.
