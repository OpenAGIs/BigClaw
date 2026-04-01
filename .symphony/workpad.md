## Plan

1. Delete the legacy Python CLI entrypoint:
   - `src/bigclaw/__main__.py`
2. Retire the stale Go compilecheck shim that still points at removed Python frozen-shim files, replacing it with an explicit empty frozen-shim contract.
3. Keep validation scoped to the affected Go legacy-shim and CLI packages, plus the tranche regression that records the Python file removal.
4. Update cutover docs so the deleted Python file is recorded as retired while preserving the historical migration notes.
5. Add a focused Go regression test that asserts:
   - the deleted Python file is absent
   - the Go-owned replacement surfaces still exist
   - the tranche is recorded as `TestTopLevelModulePurgeTranche28`
6. Run targeted validation for the touched Python syntax and Go regression package, then measure the repository Python file count delta from the current baseline.
7. Commit with a message that explicitly lists deleted Python files and added Go files/Go tests, then push the branch.

## Acceptance

- Repository Python file count decreases from the continuation baseline of `22`.
- `src/bigclaw/__main__.py` is deleted.
- The Go legacy-shim compilecheck no longer expects removed frozen Python entrypoints.
- A Go regression test covers the deletion contract and the Go replacement files for the retired Python CLI entrypoint.
- Targeted tests pass.
- Changes are committed and pushed on the working branch.

## Validation

- `cd bigclaw-go && go test ./internal/legacyshim ./cmd/bigclawctl`
- `rg --files -g '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche28$'`
- `git status --short`
- `git log -1 --stat`

## Validation Results

- `cd bigclaw-go && go test ./internal/legacyshim ./cmd/bigclawctl`
  - `ok  	bigclaw-go/internal/legacyshim	(cached)`
  - `ok  	bigclaw-go/cmd/bigclawctl	4.188s`
- `cd bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche28$'`
  - `ok  	bigclaw-go/internal/regression	(cached)`
- `rg --files -g '*.py' | wc -l`
  - `21`
