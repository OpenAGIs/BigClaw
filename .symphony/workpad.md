# BIG-GO-1227 Workpad

## Plan
- Confirm the repository-wide physical Python asset inventory, with explicit checks for `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.
- Add a lane-specific Go regression guard that keeps the repository, priority residual directories, and replacement entrypoints aligned with the zero-Python baseline for BIG-GO-1227.
- Record the lane inventory, Go replacement paths, and exact validation commands in BIG-GO-1227 report artifacts.
- Run targeted validation, capture exact command results, then commit and push to a remote issue branch.

## Acceptance
- The remaining Python asset inventory for BIG-GO-1227 is explicit and auditable.
- The repository contains no physical `.py` files, including in `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.
- A Go regression guard and lane validation artifacts are committed for BIG-GO-1227.
- Go replacement paths and exact validation commands are documented.

## Validation
- `find . -type f -name '*.py' | sort`
- `for dir in src/bigclaw tests scripts bigclaw-go/scripts; do if [ -d "$dir" ]; then find "$dir" -type f -name '*.py' | sort; fi; done`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1227(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|ReplacementPathsRemainAvailable)$'`
- `git status --short`
