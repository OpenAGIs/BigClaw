# BIG-GO-1257 Workpad

## Plan
- Reconfirm the repository-wide physical Python asset inventory, with explicit checks for `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.
- Add lane-specific regression coverage for `BIG-GO-1257` so the repository and priority residual directories stay Python-free while the supported Go replacement paths remain present.
- Add a lane report documenting the remaining Python asset inventory, the replacement paths, and the exact validation commands for this sweep.
- Run targeted validation, capture exact command results, then commit and push the lane changes.

## Acceptance
- The `BIG-GO-1257` lane has an explicit remaining Python asset inventory.
- The repository remains physically free of `.py` files, including the priority residual directories `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.
- Supported Go or shell replacement paths for the retired Python surface are documented and covered by regression tests.
- Exact validation commands and outcomes are recorded for this lane.

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
- `find . -type f -name '*.py' | sort`
- `find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1257(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
- `git status --short`
