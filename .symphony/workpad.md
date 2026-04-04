# BIG-GO-1258 Workpad

## Plan
- Reconfirm the repository-wide physical Python asset inventory, with explicit checks for `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.
- Add a lane-specific Go regression guard for `BIG-GO-1258` that preserves the zero-Python state in the repository and the priority residual directories while asserting the replacement entrypoints remain available.
- Add a lane report documenting the remaining Python asset inventory, the Go replacement paths, and the exact validation commands for this sweep.
- Run targeted validation, then record the exact commands and outcomes here before committing and pushing the lane branch.

## Acceptance
- The `BIG-GO-1258` lane has an explicit, auditable remaining Python asset inventory.
- The repository remains free of physical `.py` files, including `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.
- The retained Go or shell replacement paths for the removed Python surfaces are documented and enforced by regression coverage.
- Exact validation commands and outcomes are recorded for this lane.

## Inventory
- Repository-wide physical Python files: `0`
- `src/bigclaw/*.py`: `0` (`src/bigclaw` absent in this checkout)
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
  Result: exit `0`, no output.
- `find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
  Result: exit `0`, no output.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1258(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
  Result: exit `0`, `ok  	bigclaw-go/internal/regression	0.480s`
- `git status --short`
  Result at validation time:
  ` M .symphony/workpad.md`
  `?? bigclaw-go/docs/reports/big-go-1258-python-asset-sweep.md`
  `?? bigclaw-go/internal/regression/big_go_1258_zero_python_guard_test.go`
