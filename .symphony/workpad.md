# BIG-GO-1325 Workpad

## Plan
- Confirm the remaining physical Python asset inventory for the full repository and the priority directories `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.
- Land a lane-specific Python sweep report for `BIG-GO-1325` that records the zero-file state, the surviving Go replacement paths, and exact validation commands/results.
- Add a matching Go regression guard that asserts the repository stays Python-free and that the `BIG-GO-1325` report captures the required sweep details.
- Run targeted validation, then commit and push the branch without touching unrelated history.

## Acceptance
- The lane-specific remaining Python asset inventory is explicit.
- The lane either removes Python files or records that the repository is already Python-free and reduces the lane to regression prevention.
- Go replacement paths and exact validation commands/results are recorded in-repo.
- Repository Python file count remains `0`, with coverage for the priority directories called out in the issue.

## Validation
- `find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`
- `find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1325(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
- `git status --short`
