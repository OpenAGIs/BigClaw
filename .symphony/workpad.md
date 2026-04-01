## Plan

1. Delete the remaining Python workspace wrapper scripts that only forward into `scripts/ops/bigclawctl`:
   - `scripts/ops/bigclaw_workspace_bootstrap.py`
   - `scripts/ops/symphony_workspace_bootstrap.py`
   - `scripts/ops/symphony_workspace_validate.py`
2. Update migration and cutover docs so those removed wrappers are recorded as retired and the active operator path is the Go CLI.
3. Add a focused Go regression test that asserts:
   - the deleted workspace Python wrappers are absent
   - the Go-owned workspace replacement surfaces still exist
4. Run targeted validation for the touched Go regression package and measure the repository Python file count delta from the current baseline.
5. Commit with a message that explicitly lists deleted Python files and added Go test files, then push the branch.

## Acceptance

- Repository Python file count decreases from the continuation baseline of `58`.
- `scripts/ops/bigclaw_workspace_bootstrap.py` is deleted.
- `scripts/ops/symphony_workspace_bootstrap.py` is deleted.
- `scripts/ops/symphony_workspace_validate.py` is deleted.
- Documentation no longer presents those deleted Python wrappers as active operator entrypoints.
- A Go regression test covers the deletion contract and the Go replacement files for the removed wrappers.
- Targeted tests pass.
- Changes are committed and pushed on the working branch.

## Validation

- `rg --files -g '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche8'`
- `git status --short`
- `git log -1 --stat`
