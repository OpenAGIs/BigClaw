# BIG-GO-1279 Workpad

## Plan
- Reconfirm the repository-wide physical Python asset inventory, with explicit checks for `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.
- Confirm whether any physical Python assets remain for this lane to delete or replace.
- If the checkout is already Python-free, codify that state with a lane-specific report and Go regression guard that preserves the replacement surface.
- Run targeted validation commands, capture exact results, then commit and push the rebased branch tip to `origin/main`.

## Acceptance
- The `BIG-GO-1279` lane records the remaining Python asset inventory for the repository and priority residual directories.
- The repository remains free of physical `.py` files, including `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.
- The Go or shell replacement paths for the retired Python surface are documented and enforced by regression coverage.
- Exact validation commands and outcomes are recorded in the lane artifacts.

## Inventory
- Repository-wide physical Python files: `0`
- `src/bigclaw/*.py`: `0`
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
  - Result: no output; repository-wide Python file count is `0`.
- `find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
  - Result: no output; priority residual directories remain Python-free.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1279(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
  - Result: `ok  	bigclaw-go/internal/regression	0.479s`
