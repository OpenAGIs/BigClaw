# BIG-GO-1164 Workpad

## Plan
- Materialize the repository from `origin/main` and verify the current baseline for Python assets in this lane.
- Confirm whether the candidate `bigclaw-go/scripts/*` Python files still exist or were already removed by prior migration work.
- If the files still exist, remove them in favor of the existing Go entrypoints and validate the replacement paths.
- If the files are already gone on `main`, add a scoped regression and documentation update that locks the lane to the current Go-only baseline without widening issue scope.
- Run targeted validation, then commit and push a dedicated `BIG-GO-1164` branch.

## Acceptance
- The lane covered by `BIG-GO-1164` is validated as Go-only.
- The candidate Python paths remain absent and their Go replacement paths remain documented.
- A regression prevents reintroduction of Python files into the repository baseline.
- Validation commands and exact results are captured for closeout.

## Validation
- `find . -name '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/regression -run 'TestBIGGO1160|TestBIGGO1164|TestE2EScriptDirectoryStaysPythonFree|TestRootScriptResidualSweep'`
- `git status --short`

## Notes
- `origin/main` already materializes with `find . -name '*.py' | wc -l == 0`.
- The exact candidate list in this issue is already covered by `bigclaw-go/internal/regression/big_go_1160_script_migration_test.go`.
- This issue therefore lands incremental hardening for the zero-Python repository baseline instead of deleting files that no longer exist on the current branch.
