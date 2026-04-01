## Plan

1. Delete the isolated Python event-bus tranche that already has Go ownership:
   - `src/bigclaw/event_bus.py`
   - `tests/test_event_bus.py`
2. Remove stale package exports for the deleted Python event-bus surface from `src/bigclaw/__init__.py`.
3. Add a focused Go regression test that asserts:
   - the deleted Python files are absent
   - the Go-owned replacement surfaces still exist
4. Run targeted validation for the touched Go regression package and measure the repository Python file count delta from the current baseline.
5. Commit with a message that explicitly lists deleted Python files and added Go files/Go tests, then push the branch.

## Acceptance

- Repository Python file count decreases from the continuation baseline of `48`.
- `src/bigclaw/event_bus.py` is deleted.
- `tests/test_event_bus.py` is deleted.
- `src/bigclaw/__init__.py` no longer exports deleted Python-only event-bus symbols.
- A Go regression test covers the deletion contract and the Go replacement files for the removed event-bus surface.
- Targeted tests pass.
- Changes are committed and pushed on the working branch.

## Validation

- `rg --files -g '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche12'`
- `python3 -m py_compile src/bigclaw/__init__.py`
- `git status --short`
- `git log -1 --stat`
