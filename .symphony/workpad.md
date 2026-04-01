## Plan

1. Delete the retired Python Go-export smoke tests:
   - `tests/test_live_shadow_bundle.py`
   - `tests/test_parallel_validation_bundle.py`
   - `tests/test_validation_bundle_continuation_policy_gate.py`
2. Replace their remaining migration-only coverage with a focused Go regression contract that records the files as retired and anchors the repo-native Go regression surfaces already covering the same reports.
3. Keep validation scoped to surviving Python modules that still own adjacent report and observability behavior.
4. Update cutover docs so the deleted Python file is recorded as retired while preserving the historical migration notes.
5. Add a focused Go regression test that asserts:
   - the deleted Python files are absent
   - the Go-owned replacement surfaces still exist
   - the tranche is recorded as `TestTopLevelModulePurgeTranche27`
6. Run targeted validation for the touched Python syntax and Go regression package, then measure the repository Python file count delta from the current baseline.
7. Commit with a message that explicitly lists deleted Python files and added Go files/Go tests, then push the branch.

## Acceptance

- Repository Python file count decreases from the continuation baseline of `25`.
- `tests/test_live_shadow_bundle.py`, `tests/test_parallel_validation_bundle.py`, and `tests/test_validation_bundle_continuation_policy_gate.py` are deleted.
- A Go regression test covers the deletion contract and the Go replacement files for the retired Go-export smoke tests.
- Targeted tests pass.
- Changes are committed and pushed on the working branch.

## Validation

- `python3 -m py_compile src/bigclaw/observability.py src/bigclaw/reports.py src/bigclaw/__init__.py`
- `PYTHONPATH=src python3 -m pytest -q tests/test_observability.py tests/test_reports.py`
- `rg --files -g '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche27$'`
- `git status --short`
- `git log -1 --stat`
