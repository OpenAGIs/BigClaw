## Plan

1. Delete the next isolated Python contract tranche with existing Go ownership:
   - `src/bigclaw/connectors.py`
   - `src/bigclaw/roadmap.py`
   - `src/bigclaw/validation_policy.py`
   - `tests/test_validation_policy.py`
2. Remove stale package exports for the deleted Python modules from `src/bigclaw/__init__.py`.
3. Update cutover docs so the deleted Python sources are recorded as retired while keeping the historical migration notes intact.
4. Add a focused Go regression test that asserts:
   - the deleted Python files are absent
   - the Go-owned replacement surfaces still exist
5. Run targeted validation for the touched Go regression package and measure the repository Python file count delta from the current baseline.
6. Commit with a message that explicitly lists deleted Python files and added Go files/Go tests, then push the branch.

## Acceptance

- Repository Python file count decreases from the continuation baseline of `52`.
- `src/bigclaw/connectors.py` is deleted.
- `src/bigclaw/roadmap.py` is deleted.
- `src/bigclaw/validation_policy.py` is deleted.
- `tests/test_validation_policy.py` is deleted.
- `src/bigclaw/__init__.py` no longer exports deleted Python-only surfaces.
- A Go regression test covers the deletion contract and the Go replacement files for the removed modules.
- Targeted tests pass.
- Changes are committed and pushed on the working branch.

## Validation

- `rg --files -g '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche11'`
- `git status --short`
- `git log -1 --stat`
