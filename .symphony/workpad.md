Issue: `BIG-GO-165`
Title: `Residual tooling Python sweep L`

Plan:
- Confirm the residual tooling baseline for repo-root Python helpers and build metadata stays empty.
- Add a focused regression guard under `bigclaw-go/internal/regression` that locks down the retired tooling paths and the surviving Go/shell replacements.
- Add a lane report under `bigclaw-go/docs/reports` documenting scope, replacement paths, and exact validation commands/results for this issue.
- Run targeted validation, then commit and push the branch.

Acceptance:
- `BIG-GO-165` lands with issue-scoped evidence for residual tooling Python removal, without broad unrelated code churn.
- The regression guard fails if retired tooling Python entrypoints or root Python build helpers reappear.
- The regression guard also proves the supported replacement surface still exists.
- The lane report records the zero-inventory tooling baseline and references the issue-specific validation commands.

Validation:
- `git ls-files 'scripts/*.py' 'scripts/ops/*.py' 'setup.py' 'pyproject.toml' | sort`
- `find . -path '*/.git' -prune -o -type f \( -name '*.py' -o -name 'setup.py' -o -name 'pyproject.toml' \) -print | sed 's#^./##' | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO165(ToolingPythonPathsRemainAbsent|GoToolingReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
- `git status --short`
