# BIG-GO-1207 Workpad

## Plan
- Verify the live repository baseline for physical Python files, with explicit checks for `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.
- Add a lane-specific Go regression guard that fails if any `.py` file reappears repository-wide or inside the priority residual directories.
- Record the remaining Python asset inventory, Go replacement paths, and exact validation commands in BIG-GO-1207 report artifacts.
- Run targeted validation, capture exact results, then commit and push the branch.

## Acceptance
- The remaining Python asset inventory for BIG-GO-1207 is explicit and auditable.
- The repository contains no physical `.py` files, including in `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.
- A Go regression guard and lane validation artifacts are committed for BIG-GO-1207.
- Go replacement paths and validation commands are documented for operators.

## Validation
- `find . -name '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/regression -run 'TestBIGGO1207(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree)$'`
- `git status --short`

## Validation Results
- `find . -name '*.py' | wc -l` -> `0`
- `cd bigclaw-go && go test ./internal/regression -run 'TestBIGGO1207(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree)$'` -> `ok  	bigclaw-go/internal/regression	1.133s`
- `git status --short` -> `.symphony/workpad.md` modified; `bigclaw-go/internal/regression/big_go_1207_zero_python_guard_test.go`, `reports/BIG-GO-1207-validation.md`, and `reports/BIG-GO-1207-status.json` added before commit`
