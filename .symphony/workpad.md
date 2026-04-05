# BIG-GO-1404

## Plan
1. Inventory remaining Python assets in scope (`src/bigclaw/*.py`, `tests/*.py`, `scripts/*.py`, `bigclaw-go/scripts/*.py`, plus any other repo `.py` files).
2. Remove or shrink a safe subset within this lane, preferring direct deletion where Go replacements already exist.
3. Update any docs or references needed to point at Go equivalents and preserve validation paths.
4. Run targeted validation proving the selected assets are gone/replaced and the Go path remains usable.
5. Commit and push the scoped change.

## Acceptance
- Remaining Python assets for this lane are explicitly inventoried.
- A batch of physical Python files is removed or reduced to no-op/thin compatibility shells.
- Go replacement paths and validation commands are documented.
- Repository Python file count decreases.

## Validation
- `rg --files | rg '\\.py$'`
- Targeted repo-specific checks for any files changed in this lane.
- `git diff --stat`

## Execution Record
- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
  Result: no output
- `find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
  Result: no output
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1404(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
  Result: `ok  	bigclaw-go/internal/regression	0.615s`
