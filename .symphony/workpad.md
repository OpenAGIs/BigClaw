# BIG-GO-1255 Workpad

## Plan
- Reconfirm the repository-wide physical Python asset inventory, with explicit checks for `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.
- Add a lane-specific Go regression guard for `BIG-GO-1255` that preserves the Python-free repository baseline and asserts the retained Go replacement entrypoints still exist.
- Add lane evidence documenting the remaining Python asset inventory, the Go replacement paths, and the exact validation commands/results for this sweep.
- Run targeted validation, capture exact command outcomes, then commit and push the lane changes to `origin/main`.

## Acceptance
- The `BIG-GO-1255` lane has an explicit, auditable remaining Python asset inventory.
- The repository remains free of physical `.py` files, including `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.
- Go replacement paths for the retired Python surfaces are documented and enforced by regression coverage.
- Exact validation commands and outcomes are recorded for this lane.

## Validation
- `find . -name '*.py' -type f | wc -l`
- `for dir in src/bigclaw tests scripts bigclaw-go/scripts; do if [ -d "$dir" ]; then find "$dir" -name '*.py' -type f; fi; done`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1255(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
- `git status --short`
