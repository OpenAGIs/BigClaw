## Plan

1. Delete the standalone Python collaboration helper module:
   - `src/bigclaw/collaboration.py`
2. Move the collaboration thread dataclasses, builders, and render helpers into `src/bigclaw/observability.py`, which already owns the adjacent audit/comment/decision data surface.
3. Preserve `bigclaw.collaboration` import compatibility through package exports after the file deletion.
4. Update cutover docs so the deleted Python file is recorded as retired while preserving the historical migration notes.
5. Add a focused Go regression test that asserts:
   - the deleted Python file is absent
   - the Go-owned replacement surfaces still exist
   - the tranche is recorded as `TestTopLevelModulePurgeTranche23`
6. Run targeted validation for the touched Python syntax and Go regression package, then measure the repository Python file count delta from the current baseline.
7. Commit with a message that explicitly lists deleted Python files and added Go files/Go tests, then push the branch.

## Acceptance

- Repository Python file count decreases from the continuation baseline of `30`.
- `src/bigclaw/collaboration.py` is deleted.
- `src/bigclaw/observability.py` still provides the collaboration surface, and `bigclaw.collaboration` imports still resolve without the deleted file.
- A Go regression test covers the deletion contract and the Go replacement files for the removed collaboration helper.
- Targeted tests pass.
- Changes are committed and pushed on the working branch.

## Validation

- `python3 -m py_compile src/bigclaw/observability.py src/bigclaw/reports.py src/bigclaw/__init__.py`
- `PYTHONPATH=src python3 -m pytest -q tests/test_observability.py tests/test_reports.py tests/test_repo_collaboration.py`
- `rg --files -g '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche23$'`
- `git status --short`
- `git log -1 --stat`
