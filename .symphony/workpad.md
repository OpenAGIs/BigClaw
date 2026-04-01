## Plan

1. Delete the retired Python collaboration helper test file:
   - `tests/test_repo_collaboration.py`
2. Replace its remaining migration-only coverage with a focused Go regression contract that records the file as retired and anchors the Go-owned replacement surfaces.
3. Keep validation scoped to surviving Python modules that still own the collaboration behavior exercised by nearby tests.
4. Update cutover docs so the deleted Python file is recorded as retired while preserving the historical migration notes.
5. Add a focused Go regression test that asserts:
   - the deleted Python file is absent
   - the Go-owned replacement surfaces still exist
   - the tranche is recorded as `TestTopLevelModulePurgeTranche26`
6. Run targeted validation for the touched Python syntax and Go regression package, then measure the repository Python file count delta from the current baseline.
7. Commit with a message that explicitly lists deleted Python files and added Go files/Go tests, then push the branch.

## Acceptance

- Repository Python file count decreases from the continuation baseline of `26`.
- `tests/test_repo_collaboration.py` is deleted.
- A Go regression test covers the deletion contract and the Go replacement files for the retired collaboration helper test.
- Targeted tests pass.
- Changes are committed and pushed on the working branch.

## Validation

- `python3 -m py_compile src/bigclaw/observability.py src/bigclaw/reports.py src/bigclaw/__init__.py`
- `PYTHONPATH=src python3 -m pytest -q tests/test_observability.py tests/test_reports.py`
- `rg --files -g '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche26$'`
- `git status --short`
- `git log -1 --stat`
