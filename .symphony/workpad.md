# BIG-GO-1249 Workpad

## Plan
- Reconfirm the repository-wide physical Python asset inventory, with explicit checks for `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.
- Add a lane-specific Go regression guard for `BIG-GO-1249` that keeps the repository and the priority residual directories Python-free while asserting the replacement entrypoints still exist.
- Add a lane report documenting the remaining Python asset inventory, the Go replacement paths, and the exact validation commands/results for this sweep.
- Run targeted validation, capture exact command outcomes, then commit and push the lane changes to `origin/main`.

## Acceptance
- The `BIG-GO-1249` lane has an explicit, auditable remaining Python asset inventory.
- The repository remains free of physical `.py` files, including the priority directories `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.
- The retained Go or shell replacement paths for the removed Python assets are documented and enforced by regression coverage.
- Exact validation commands and outcomes are recorded for this lane.

## Inventory
- Repository-wide physical Python files: `0`
- `src/bigclaw/*.py`: `0` (`src/bigclaw` is absent)
- `tests/*.py`: `0` (`tests` is absent)
- `scripts/*.py`: `0`
- `bigclaw-go/scripts/*.py`: `0`

## Go Replacement Paths
- `scripts/ops/bigclawctl`
- `scripts/dev_bootstrap.sh`
- `bigclaw-go/cmd/bigclawctl/main.go`
- `bigclaw-go/cmd/bigclawd/main.go`
- `bigclaw-go/scripts/e2e/run_all.sh`

## Validation
- `find . -type f -name '*.py' | sort`
- `find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1249(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
- `git status --short`
