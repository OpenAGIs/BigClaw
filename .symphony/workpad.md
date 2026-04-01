## Plan

1. Delete the isolated Python design-system tranche with Go-owned console/product replacements:
   - `src/bigclaw/design_system.py`
   - `tests/test_design_system.py`
2. Remove stale package exports for the deleted Python design-system surface from `src/bigclaw/__init__.py`.
3. Update cutover docs so the deleted Python source is recorded as retired while preserving the historical migration notes.
4. Add a focused Go regression test that asserts:
   - the deleted Python files are absent
   - the Go-owned replacement surfaces still exist
5. Run targeted validation for the touched Go regression package and measure the repository Python file count delta from the current baseline.
6. Commit with a message that explicitly lists deleted Python files and added Go files/Go tests, then push the branch.

## Acceptance

- Repository Python file count decreases from the continuation baseline of `42`.
- `src/bigclaw/design_system.py` is deleted.
- `tests/test_design_system.py` is deleted.
- `src/bigclaw/__init__.py` no longer exports deleted Python-only design-system symbols.
- A Go regression test covers the deletion contract and the Go replacement files for the removed design-system surface.
- Targeted tests pass.
- Changes are committed and pushed on the working branch.

## Validation

- `rg --files -g '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche15'`
- `python3 -m py_compile src/bigclaw/__init__.py`
- `git status --short`
- `git log -1 --stat`
