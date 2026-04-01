## Plan

1. Delete the now-unreferenced Python compatibility helper module:
   - `src/bigclaw/legacy_shim.py`
2. Update the Go-side legacy compile-check surface so it no longer expects that deleted Python file in the frozen list.
3. Add a focused Go regression test that asserts:
   - the deleted Python helper is absent
   - the Go-owned replacement surfaces still exist
4. Run targeted validation for the touched Go packages and measure the repository Python file count delta from the current baseline.
5. Commit with a message that explicitly lists deleted Python files and added Go files/Go tests, then push the branch.

## Acceptance

- Repository Python file count decreases from the continuation baseline of `53`.
- `src/bigclaw/legacy_shim.py` is deleted.
- Go compile-check expectations no longer require that deleted Python file.
- A Go regression test covers the deletion contract and the Go replacement files for the removed helper.
- Targeted tests pass.
- Changes are committed and pushed on the working branch.

## Validation

- `rg --files -g '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/legacyshim ./cmd/bigclawctl ./internal/regression -run 'Test(FrozenCompileCheckFilesUsesFrozenShimList|CompileCheckRunsPyCompileAgainstFrozenShimList|RunLegacyPythonCompileCheckJSONOutputDoesNotEscapeArrowTokens|TopLevelModulePurgeTranche10)'`
- `git status --short`
- `git log -1 --stat`
