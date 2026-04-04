# BIG-GO-1259 Workpad

## Plan
- Reconfirm the repository-wide physical Python asset inventory, with explicit checks for `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.
- Add a lane-specific Go regression guard for `BIG-GO-1259` that keeps the repository and the priority residual directories Python-free while asserting the replacement entrypoints still exist.
- Add a lane report documenting the remaining Python asset inventory, the Go replacement paths, and the exact validation commands/results for this sweep.
- Run targeted validation, capture exact command outcomes, then commit and push the lane changes to the remote issue branch.

## Acceptance
- The `BIG-GO-1259` lane has an explicit, auditable remaining Python asset inventory.
- The repository remains free of physical `.py` files, including the priority directories `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.
- The retained Go or shell replacement paths for the removed Python assets are documented and enforced by regression coverage.
- Exact validation commands and outcomes are recorded for this lane.

## Inventory
- Repository-wide physical Python files: `0`
- `src/bigclaw/*.py`: `0` (`src/bigclaw` is absent)
- `tests/*.py`: `0`
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
- Result: no output; exit `0`.
- `find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
- Result: no output; exit `0`.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1259(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
- Result: `ok  	bigclaw-go/internal/regression	0.262s`
- `git status --short`
