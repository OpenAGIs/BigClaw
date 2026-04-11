Issue: BIG-GO-222
Title: Residual tests Python sweep AI

Plan
- Audit the existing residual-test Python sweep pattern and confirm which repo-native directories and replacement artifacts this lane should harden.
- Add a lane-specific Go regression guard under `bigclaw-go/internal/regression` covering the repository-wide zero-Python baseline, the remaining Python-heavy test directories, and the retained Go/native replacement paths.
- Add a lane report under `bigclaw-go/docs/reports` that captures the scoped sweep, the retained replacement evidence, and the exact validation commands/results for this issue.
- Run the targeted sweep commands and regression tests, record exact results, then commit and push the scoped lane changes.

Acceptance
- `.symphony/workpad.md` exists and reflects the scoped `BIG-GO-222` plan before code changes.
- A new `BIG-GO-222` regression test file exists and passes.
- A new `BIG-GO-222` report exists and is asserted by the regression test.
- Validation evidence records exact commands and results for the repository-wide Python sweep, the scoped residual test directory sweep, and the targeted Go regression run.
- Changes remain scoped to the workpad, the `BIG-GO-222` regression guard, the `BIG-GO-222` report, and git metadata from commit/push.

Validation
- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find tests reports bigclaw-go/docs/reports bigclaw-go/internal/regression bigclaw-go/internal/migration -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO222(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
