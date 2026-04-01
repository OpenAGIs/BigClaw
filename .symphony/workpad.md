# BIG-GO-1059

## Plan
- Inspect current workspace bootstrap entrypoints under `scripts/ops/` and all references from docs, CI, workflows, and hooks.
- Remove workspace bootstrap Python compatibility wrappers and route callers to the Go or shell bootstrap entrypoints only.
- Update affected documentation and automation so no bootstrap path points at the deleted Python files.
- Run targeted validation covering bootstrap entrypoints, references, and Python file-count regression checks.
- Commit the scoped changes and push the issue branch to the remote.

## Acceptance
- `scripts/ops/*workspace*.py` compatibility wrappers targeted by this issue are deleted rather than only documented away.
- README, workflows, hooks, and CI no longer reference deleted Python workspace bootstrap entrypoints.
- Workspace bootstrap still resolves through supported Go or shell entrypoints.
- Repository `.py` count decreases versus the pre-change state.

## Validation
- Use repository search to confirm no remaining references to deleted workspace bootstrap Python files.
- Run targeted tests for regression coverage related to repo links, orchestration, or bootstrap entrypoints if present.
- Record exact commands and results in the final report.
