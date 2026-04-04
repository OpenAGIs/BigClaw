# BIG-GO-1232 Workpad

## Plan
- Reconfirm the repository-wide physical Python asset inventory, with explicit checks for `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.
- Add a lane-specific Go regression guard that keeps the repository and priority residual directories Python-free for BIG-GO-1232.
- Record the lane inventory, Go replacement paths, and exact validation commands in BIG-GO-1232 report artifacts.
- Run targeted validation, capture exact command results, then commit and push to `origin/main`.

## Acceptance
- The remaining Python asset inventory for BIG-GO-1232 is explicit and auditable.
- The repository contains no physical `.py` files, including in `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.
- A Go regression guard and lane validation artifacts are committed for BIG-GO-1232.
- Go replacement paths and exact validation commands are documented.

## Validation
- `find . -name '*.py' -type f | wc -l`
- `for dir in src/bigclaw tests scripts bigclaw-go/scripts; do if [ -d "$dir" ]; then find "$dir" -name '*.py' -type f; fi; done`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1232(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree)$'`
- `git status --short`
