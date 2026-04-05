# BIG-GO-1344

## Plan
1. Record the live Python asset inventory for this checkout, with explicit coverage for `src/bigclaw/*.py`, `tests/*.py`, `scripts/*.py`, and `bigclaw-go/scripts/*.py`.
2. Add lane-scoped reporting artifacts that document the already-zero Python baseline and the current Go replacement paths for the retired Python surfaces.
3. Add a regression guard under `bigclaw-go/internal/regression` that fails if any physical `.py` file reappears repository-wide or under the priority residual directories.
4. Run targeted inventory and regression commands, then capture the exact commands and outcomes in the validation report.
5. Commit the lane changes and push them to `origin/main`.

## Acceptance
- Produce a clear inventory of remaining Python assets relevant to this lane.
- Document the zero-Python baseline when no removable physical `.py` assets remain in this checkout.
- Document the Go replacement path(s) and validation commands.
- Keep the repository Python file count at zero and harden it with regression coverage.

## Validation
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1344 -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1344/src/bigclaw /Users/openagi/code/bigclaw-workspaces/BIG-GO-1344/tests /Users/openagi/code/bigclaw-workspaces/BIG-GO-1344/scripts /Users/openagi/code/bigclaw-workspaces/BIG-GO-1344/bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1344/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1344(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
