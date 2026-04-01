## Plan

1. Delete the standalone Python audit-events helper module:
   - `src/bigclaw/audit_events.py`
2. Move the audit event constants, spec registry, and required-field helper into `src/bigclaw/observability.py`, which already owns the adjacent audit data surface.
3. Preserve `bigclaw.audit_events` import compatibility through package exports after the file deletion.
4. Update cutover docs so the deleted Python file is recorded as retired while preserving the historical migration notes.
5. Add a focused Go regression test that asserts:
   - the deleted Python file is absent
   - the Go-owned replacement surfaces still exist
   - the tranche is recorded as `TestTopLevelModulePurgeTranche22`
6. Run targeted validation for the touched Python syntax and Go regression package, then measure the repository Python file count delta from the current baseline.
7. Commit with a message that explicitly lists deleted Python files and added Go files/Go tests, then push the branch.

## Acceptance

- Repository Python file count decreases from the continuation baseline of `31`.
- `src/bigclaw/audit_events.py` is deleted.
- `src/bigclaw/observability.py` still provides the audit event registry surface, and `bigclaw.audit_events` imports still resolve without the deleted file.
- A Go regression test covers the deletion contract and the Go replacement files for the removed audit-events helper.
- Targeted tests pass.
- Changes are committed and pushed on the working branch.

## Validation

- `python3 -m py_compile src/bigclaw/observability.py src/bigclaw/reports.py src/bigclaw/runtime.py src/bigclaw/__init__.py`
- `PYTHONPATH=src python3 -m pytest -q tests/test_observability.py`
- `rg --files -g '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche22$'`
- `git status --short`
- `git log -1 --stat`
