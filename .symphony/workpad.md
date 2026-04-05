# BIG-GO-1264 Workpad

## Plan
- Reconfirm the repository-wide physical Python asset inventory, with explicit checks for `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.
- Add a lane-specific Go regression guard for `BIG-GO-1264` that keeps the repository and the priority residual directories Python-free while asserting the retained Go or shell replacement entrypoints still exist.
- Add a lane report and validation/status artifacts documenting the remaining Python asset inventory, replacement paths, and exact command results for this sweep.
- Run targeted validation, capture exact outcomes, then commit and push the lane changes to the remote branch for this workspace.

## Acceptance
- The `BIG-GO-1264` lane has an explicit, auditable remaining Python asset inventory.
- The repository remains free of physical `.py` files, including `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.
- Go or shell replacement paths for the retired Python entrypoints are documented and enforced by regression coverage.
- Exact validation commands and results are recorded for this lane.

## Validation
- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1264(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
- `git status --short`
