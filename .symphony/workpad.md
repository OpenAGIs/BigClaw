# BIG-GO-1293 Workpad

## Plan
- Reconfirm the repository-wide physical Python asset inventory, with explicit checks for `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.
- Add a lane-specific Go regression guard for `BIG-GO-1293` that preserves the Python-free repository state and verifies the retained Go or shell replacement entrypoints still exist.
- Record the lane sweep in durable artifacts under `bigclaw-go/docs/reports` and `reports`, including the remaining Python asset inventory, replacement paths, and exact validation commands.
- Run the targeted validation commands for this lane, capture the exact results, then commit and push the branch tip to `origin/main`.

## Acceptance
- The `BIG-GO-1293` lane has an explicit remaining Python asset inventory for the repository and priority residual directories.
- The repository remains free of physical `.py` files, including `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.
- The Go or shell replacement paths for the retired Python surface are documented and enforced by regression coverage.
- Exact validation commands and outcomes are recorded in the lane artifacts.

## Inventory
- Repository-wide physical Python files: `0`
- `src/bigclaw/*.py`: `0` (`src/bigclaw` is absent)
- `tests/*.py`: `0`
- `scripts/*.py`: `0`
- `bigclaw-go/scripts/*.py`: `0`

## Replacement Paths
- `scripts/ops/bigclawctl`
- `scripts/dev_bootstrap.sh`
- `bigclaw-go/cmd/bigclawctl/main.go`
- `bigclaw-go/cmd/bigclawd/main.go`
- `bigclaw-go/scripts/e2e/run_all.sh`

## Validation
- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1293(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
