## Plan

1. Delete the standalone Python run-detail helper module:
   - `src/bigclaw/run_detail.py`
2. Move the shared run-detail dataclasses and rendering helpers into `src/bigclaw/reports.py`, which already owns the main task-run detail page.
3. Update `src/bigclaw/evaluation.py` to import the shared helpers from `src/bigclaw/reports.py` instead of the deleted file.
4. Update cutover docs so the deleted Python file is recorded as retired while preserving the historical migration notes.
5. Add a focused Go regression test that asserts:
   - the deleted Python file is absent
   - the Go-owned replacement surfaces still exist
   - the tranche is recorded as `TestTopLevelModulePurgeTranche19`
6. Run targeted validation for the touched Python syntax and Go regression package, then measure the repository Python file count delta from the current baseline.
7. Commit with a message that explicitly lists deleted Python files and added Go files/Go tests, then push the branch.

## Acceptance

- Repository Python file count decreases from the continuation baseline of `34`.
- `src/bigclaw/run_detail.py` is deleted.
- `src/bigclaw/reports.py` and `src/bigclaw/evaluation.py` still render the existing run-detail and replay-detail surfaces without importing the deleted file.
- A Go regression test covers the deletion contract and the Go replacement files for the removed run-detail helper.
- Targeted tests pass.
- Changes are committed and pushed on the working branch.

## Validation

- `python3 -m py_compile src/bigclaw/reports.py src/bigclaw/evaluation.py src/bigclaw/observability.py`
- `PYTHONPATH=src python3 -m pytest -q tests/test_observability.py tests/test_evaluation.py`
- `rg --files -g '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche19$'`
- `git status --short`
- `git log -1 --stat`
