# BIG-GO-1243 Workpad

## Plan
- Confirm the remaining physical Python asset inventory for the repository, with explicit checks for `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.
- Add a lane-specific Go regression guard for BIG-GO-1243 that preserves the current zero-Python repository state and keeps the priority residual directories empty.
- Validate concrete Go or shell replacement paths that now cover the historical Python entrypoint surface for this lane.
- Run the targeted inventory and regression commands, capture exact results, then commit and push to `origin/main`.

## Acceptance
- The BIG-GO-1243 lane records an explicit remaining Python asset inventory.
- The repository Python file count stays at `0`, including under `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.
- A Go regression guard for BIG-GO-1243 is committed.
- Go replacement paths and exact validation commands are documented.

## Inventory
- Repository-wide physical Python files: `0`
- `src/bigclaw/*.py`: `0` (`src/bigclaw` is absent)
- `tests/*.py`: `0`
- `scripts/*.py`: `0`
- `bigclaw-go/scripts/*.py`: `0`

## Go Replacement Paths
- `scripts/ops/bigclawctl`
- `bigclaw-go/cmd/bigclawctl/main.go`
- `scripts/dev_bootstrap.sh`
- `bigclaw-go/internal/bootstrap/bootstrap.go`

## Validation
- `find . -name '*.py' -type f | wc -l`
- `for dir in src/bigclaw tests scripts bigclaw-go/scripts; do if [ -d "$dir" ]; then find "$dir" -name '*.py' -type f; fi; done`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1243(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|ReplacementPathsRemainAvailable)$'`
- `git status --short`
