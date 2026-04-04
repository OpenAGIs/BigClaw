# BIG-GO-1250 Workpad

## Plan
- Confirm the repository-wide physical Python asset inventory, with explicit checks for `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.
- Add a lane-specific Go regression guard that keeps the repository and priority residual directories Python-free for BIG-GO-1250 and verifies the retained Go replacement paths still exist.
- Record the lane inventory, Go replacement paths, and exact validation commands in BIG-GO-1250 report artifacts.
- Run targeted validation, capture exact command results, then commit and push to `origin/main`.

## Acceptance
- The remaining Python asset inventory for BIG-GO-1250 is explicit and auditable.
- The repository contains no physical `.py` files, including in `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.
- A Go regression guard and lane validation artifacts are committed for BIG-GO-1250.
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
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1250(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|ReplacementPathsRemainAvailable)$'`
- `git status --short`
