# BIG-GO-1525 Workpad

## Plan
- Recover the issue workspace on a valid branch and capture the repository-wide `.py` baseline.
- Inspect reporting and observability Python modules plus direct tests/importers to identify a safe, scoped deletion sweep.
- Remove only the confirmed residual Python files for this issue.
- Record exact removed-file evidence and before/after `.py` counts in a repo report.
- Run targeted validation commands covering the touched surface and confirm the repository `.py` count decreased.
- Commit the change on `BIG-GO-1525` and push to `origin/BIG-GO-1525`.

## Acceptance
- Repository `.py` file count decreases from the captured baseline.
- Removed files are limited to reporting/observability Python residuals for this issue.
- A ledger in the repository records the exact removed file paths and before/after counts.
- Targeted validation commands complete successfully and are captured with exact commands and results.
- Changes are committed and pushed on the issue branch.

## Validation
- `find . -type f -name '*.py' | wc -l`
- `rg -n "bigclaw\\.(reports|observability)" src tests scripts bigclaw-go`
- Targeted test commands chosen after dependency inspection.
- `git status --short`
- `git diff --stat`
