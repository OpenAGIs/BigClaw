# BIG-GO-1235 Workpad

## Plan
- Confirm the repository-wide physical Python asset inventory, with explicit checks for `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.
- Remove stale root documentation that still points operators at the retired `legacy-python compile-check` flow.
- Add a lane-specific Go regression guard that preserves the zero-Python baseline and the Go-only README guidance.
- Record the lane inventory, Go replacement paths, and exact validation commands in BIG-GO-1235 report artifacts.
- Run targeted validation, capture exact command results, then commit and push to `origin/main`.

## Acceptance
- The remaining Python asset inventory for BIG-GO-1235 is explicit and auditable.
- The repository contains no physical `.py` files, including in `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.
- Root documentation no longer presents the retired `legacy-python compile-check` path as an active operator workflow.
- A Go regression guard and lane validation artifacts are committed for BIG-GO-1235.
- Go replacement paths and exact validation commands are documented.

## Validation
- `find . -name '*.py' -type f | wc -l`
- `for dir in src/bigclaw tests scripts bigclaw-go/scripts; do if [ -d "$dir" ]; then find "$dir" -name '*.py' -type f; fi; done`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1235(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|ReadmeStaysGoOnly)$'`
- `git status --short`

## Validation Results
- `find . -name '*.py' -type f | wc -l` -> `0`
- `for dir in src/bigclaw tests scripts bigclaw-go/scripts; do if [ -d "$dir" ]; then find "$dir" -name '*.py' -type f; fi; done` -> `<empty>`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1235(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|ReadmeStaysGoOnly)$'` -> `ok  	bigclaw-go/internal/regression	3.219s`

## Residual Risk
- The live workspace baseline is already at a repository-wide Python file count of `0`, so BIG-GO-1235 can only harden and document the Go-only state rather than reduce the count numerically.
