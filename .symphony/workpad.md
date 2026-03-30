# BIG-GO-1005 Workpad

## Plan
- Inventory the remaining `src/bigclaw/*.py` batch and capture the current repo-wide Python file count baseline.
- Trace package exports, tests, scripts, and CLI entrypoints to separate safe deletions from modules that still provide live coverage or operator-facing compatibility.
- Remove or consolidate the safe residual Python modules in this batch, keeping changes scoped to `src/bigclaw/**` and any directly required test/report updates.
- Record a batch report with delete/replace/keep rationale and the exact before/after Python file count impact.
- Run targeted validation for the touched surfaces, then commit and push the branch.

## Acceptance
- Produce the explicit residual Python file list for this `src/bigclaw/**` batch.
- Reduce the number of Python files in this batch where the repo state makes that safe.
- Document why each affected module was deleted, replaced, or kept.
- Report the effect on the total Python file count in the repository.

## Validation
- Use `rg --files src/bigclaw -g '*.py'` and `rg --files -g '*.py' | wc -l` before and after changes to capture batch inventory and repo-wide file counts.
- Run targeted Python validation against touched modules and their tests with exact commands captured in the report.
- Run any additional package or import smoke checks required by the final shape of `src/bigclaw/**`.
- Capture `git status --short` and the final commit hash after commit/push.
