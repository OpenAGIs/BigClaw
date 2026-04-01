## Plan

1. Delete the standalone Python governance helper module:
   - `src/bigclaw/governance.py`
2. Move the scope-freeze governance dataclasses and renderer into `src/bigclaw/planning.py`, which is the only remaining Python consumer.
3. Preserve `bigclaw.governance` import compatibility through package exports after the file deletion.
4. Update cutover docs so the deleted Python file is recorded as retired while preserving the historical migration notes.
5. Add a focused Go regression test that asserts:
   - the deleted Python file is absent
   - the Go-owned replacement surfaces still exist
   - the tranche is recorded as `TestTopLevelModulePurgeTranche20`
6. Run targeted validation for the touched Python syntax and Go regression package, then measure the repository Python file count delta from the current baseline.
7. Commit with a message that explicitly lists deleted Python files and added Go files/Go tests, then push the branch.

## Acceptance

- Repository Python file count decreases from the continuation baseline of `33`.
- `src/bigclaw/governance.py` is deleted.
- `src/bigclaw/planning.py` still provides the scope-freeze governance surface, and `bigclaw.governance` imports still resolve without the deleted file.
- A Go regression test covers the deletion contract and the Go replacement files for the removed governance helper.
- Targeted tests pass.
- Changes are committed and pushed on the working branch.

## Validation

- `python3 -m py_compile src/bigclaw/planning.py src/bigclaw/__init__.py`
- `PYTHONPATH=src python3 -m pytest -q tests/test_planning.py`
- `rg --files -g '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche20$'`
- `git status --short`
- `git log -1 --stat`
