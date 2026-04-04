# BIG-GO-1256 Workpad

## Plan

1. Audit repository-wide physical Python assets, with priority on `src/bigclaw`,
   `tests`, `scripts`, and `bigclaw-go/scripts`.
2. Record the lane inventory, Go replacement paths, and validation evidence in
   lane-specific report artifacts.
3. Add a regression guard that keeps the repository and priority directories
   Python-free and verifies the replacement paths remain available.
4. Run targeted validation commands, capture exact results, then commit and push
   the lane changes.

## Acceptance

- Remaining Python asset inventory for the lane is explicitly recorded.
- The lane either removes residual `.py` files or, if the baseline is already
  zero, hardens and documents that Go-only state.
- Go replacement paths and verification commands are recorded.
- Targeted tests run successfully and exact commands plus results are preserved.

## Validation

- `find . -type f -name '*.py' | wc -l`
- `for dir in src/bigclaw tests scripts bigclaw-go/scripts; do if [ -d "$dir" ]; then find "$dir" -type f -name '*.py' | sort; fi; done`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1256(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|ReplacementPathsRemainAvailable)$'`
