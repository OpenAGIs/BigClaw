## Plan

1. Delete the standalone Python memory helper module:
   - `src/bigclaw/memory.py`
2. Move the memory-pattern dataclasses and store into `src/bigclaw/models.py`, which already owns the adjacent task domain types they persist.
3. Preserve `bigclaw.memory` import compatibility through package exports after the file deletion.
4. Update cutover docs so the deleted Python file is recorded as retired while preserving the historical migration notes.
5. Add a focused Go regression test that asserts:
   - the deleted Python file is absent
   - the Go-owned replacement surfaces still exist
   - the tranche is recorded as `TestTopLevelModulePurgeTranche24`
6. Run targeted validation for the touched Python syntax and Go regression package, then measure the repository Python file count delta from the current baseline.
7. Commit with a message that explicitly lists deleted Python files and added Go files/Go tests, then push the branch.

## Acceptance

- Repository Python file count decreases from the continuation baseline of `29`.
- `src/bigclaw/memory.py` is deleted.
- `src/bigclaw/models.py` still provides the memory-pattern surface, and `bigclaw.memory` imports still resolve without the deleted file.
- A Go regression test covers the deletion contract and the Go replacement files for the removed memory helper.
- Targeted tests pass.
- Changes are committed and pushed on the working branch.

## Validation

- `python3 -m py_compile src/bigclaw/models.py src/bigclaw/__init__.py`
- `PYTHONPATH=src python3 -m pytest -q tests/test_memory.py`
- `rg --files -g '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche24$'`
- `git status --short`
- `git log -1 --stat`
