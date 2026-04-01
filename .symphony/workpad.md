## Plan

1. Delete the retired Python helper test files:
   - `tests/test_risk.py`
   - `tests/test_memory.py`
2. Replace their remaining migration-only coverage with a focused Go regression contract that records the files as retired and anchors the Go-owned replacement surfaces.
3. Keep validation scoped to surviving Python modules that still own the compatibility behavior exercised by nearby tests.
4. Update cutover docs so the deleted Python file is recorded as retired while preserving the historical migration notes.
5. Add a focused Go regression test that asserts:
   - the deleted Python files are absent
   - the Go-owned replacement surfaces still exist
   - the tranche is recorded as `TestTopLevelModulePurgeTranche25`
6. Run targeted validation for the touched Python syntax and Go regression package, then measure the repository Python file count delta from the current baseline.
7. Commit with a message that explicitly lists deleted Python files and added Go files/Go tests, then push the branch.

## Acceptance

- Repository Python file count decreases from the continuation baseline of `28`.
- `tests/test_risk.py` and `tests/test_memory.py` are deleted.
- A Go regression test covers the deletion contract and the Go replacement files for the retired Python helper tests.
- Targeted tests pass.
- Changes are committed and pushed on the working branch.

## Validation

- `python3 -m py_compile src/bigclaw/models.py src/bigclaw/runtime.py src/bigclaw/__init__.py`
- `PYTHONPATH=src python3 -m pytest -q tests/test_models.py tests/test_runtime_matrix.py tests/test_scheduler.py`
- `rg --files -g '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche25$'`
- `git status --short`
- `git log -1 --stat`
