# BIG-GO-1188 Workpad

## Plan
- Verify the live repository baseline for physical Python files, with extra focus on `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.
- Add a narrow Go regression guard that fails if any `.py` file reappears anywhere in the repository or in the lane's priority directories.
- Record the remaining Python asset inventory, Go replacement paths, and exact validation commands in lane-specific report artifacts.
- Run targeted validation, capture exact results, then commit and push the branch.

## Acceptance
- The remaining Python asset inventory for this lane is explicit and auditable.
- The repository contains no physical `.py` files, including in `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.
- A Go regression guard and lane validation artifacts are committed for BIG-GO-1188.
- Go replacement paths and validation commands are documented for operators.

## Validation
- `find . -name '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/regression -run 'TestBIGGO1188(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree)$'`
- `git status --short`

## Validation Results
- `find . -name '*.py' | wc -l` -> `0`
- `cd bigclaw-go && go test ./internal/regression -run 'TestBIGGO1188(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree)$'` -> `ok  	bigclaw-go/internal/regression	0.493s`
- `git status --short` -> `.symphony/workpad.md` modified and `bigclaw-go/internal/regression/big_go_1188_zero_python_guard_test.go`, `reports/BIG-GO-1188-validation.md`, `reports/BIG-GO-1188-status.json` added before commit

## Residual Risk
- If the workspace baseline is already at a repository-wide `.py` count of `0`, this lane can only harden and document the zero-Python state rather than reduce the count numerically.

---

# Archived: BIG-GO-1184 Workpad

## Plan
- Verify the repository-wide Python asset baseline, with explicit inventory
  checks for `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.
- Add lane-specific Go regression coverage that locks the repository to a
  zero-`.py` state and confirms the expected Go or shell replacement surface
  still exists.
- Record lane validation and status artifacts with the remaining Python asset
  list, replacement paths, exact commands, and results.
- Run targeted validation, then commit and push the branch.

## Acceptance
- The lane records the remaining physical Python asset inventory for the
  priority directories and the repository overall.
- The priority residual directories remain free of physical `.py` files.
- The lane points operators to the current Go or shell replacement paths for
  the retired Python surfaces.
- Targeted regression and audit artifacts are committed for BIG-GO-1184.

## Validation
- `find . -name '*.py' | wc -l`
- `for dir in src/bigclaw tests scripts bigclaw-go/scripts; do if [ -d "$dir" ]; then find "$dir" -name '*.py' | sort; else printf '[absent] %s\n' "$dir"; fi; done`
- `cd bigclaw-go && go test ./internal/regression -run 'TestBIGGO1184(RepositoryHasNoPythonFiles|PriorityResidualInventoryAndReplacementSurface)$'`
- `git status --short`

## Validation Results
- `find . -name '*.py' | wc -l` -> `0`
- `for dir in src/bigclaw tests scripts bigclaw-go/scripts; do if [ -d "$dir" ]; then find "$dir" -name '*.py' | sort; else printf '[absent] %s\n' "$dir"; fi; done` -> `[absent] src/bigclaw`, `[absent] tests`
- `cd bigclaw-go && go test ./internal/regression -run 'TestBIGGO1184(RepositoryHasNoPythonFiles|PriorityResidualInventoryAndReplacementSurface)$'` -> `ok   bigclaw-go/internal/regression  0.452s`
- `git status --short` -> `M .symphony/workpad.md`, `?? bigclaw-go/internal/regression/big_go_1184_python_residual_inventory_test.go`, `?? reports/BIG-GO-1184-status.json`, `?? reports/BIG-GO-1184-validation.md`

## Residual Risk
- The repository is already at `0` physical `.py` files, so this lane
  preserves and proves the zero-Python baseline instead of reducing the count
  further.
