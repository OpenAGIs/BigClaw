Issue: BIG-GO-215
Title: Residual tooling Python sweep Q

Plan
- Audit the current zero-Python baseline around tooling, build helpers, and dev utilities to identify the directories and native replacement paths that belong in this lane.
- Add a lane-specific Go regression guard under `bigclaw-go/internal/regression` that locks the repository-wide zero-Python state and the chosen tooling/dev utility directories.
- Add a lane report under `bigclaw-go/docs/reports` documenting the sweep scope, replacement paths, and exact validation commands/results.
- Run the targeted repository sweep and regression tests, record exact commands/results, then commit and push the scoped changes.

Acceptance
- `.symphony/workpad.md` exists and reflects the scoped plan before code changes.
- A new `BIG-GO-215` regression test file exists and passes.
- A new `BIG-GO-215` report exists and is referenced by the regression test assertions.
- Validation evidence records exact commands and results for the repository-wide Python sweep, the scoped tooling/dev utility directory sweep, and the targeted Go regression test.
- Changes stay limited to the lane workpad, the new regression guard, the new report, and git metadata from commit/push.

Validation
- `find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`
- `find .github .githooks scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO215(RepositoryHasNoPythonFiles|ToolingDirectoriesStayPythonFree|NativeToolingReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
