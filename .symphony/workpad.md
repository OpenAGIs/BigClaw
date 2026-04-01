## Plan

1. Delete the standalone Python deprecation helper:
   - `src/bigclaw/deprecation.py`
2. Inline the minimal legacy-warning helper into the two remaining Python consumers:
   - `src/bigclaw/__main__.py`
   - `src/bigclaw/runtime.py`
3. Update cutover docs so the deleted Python file is recorded as retired while preserving the migration-compatibility contract.
4. Add a focused Go regression test that asserts:
   - the deleted Python file is absent
   - the Go-owned replacement and compatibility-contract surfaces still exist
5. Run targeted validation for the touched Python syntax and Go regression package, then measure the repository Python file count delta from the current baseline.
6. Commit with a message that explicitly lists deleted Python files and added Go files/Go tests, then push the branch.

## Acceptance

- Repository Python file count decreases from the continuation baseline of `35`.
- `src/bigclaw/deprecation.py` is deleted.
- `src/bigclaw/__main__.py` and `src/bigclaw/runtime.py` still emit the same migration-only warning contract without importing the deleted file.
- A Go regression test covers the deletion contract and the Go replacement files for the removed deprecation helper.
- Targeted tests pass.
- Changes are committed and pushed on the working branch.

## Validation

- `python3 -m py_compile src/bigclaw/__main__.py src/bigclaw/runtime.py`
- `rg --files -g '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche18'`
- `git status --short`
- `git log -1 --stat`
