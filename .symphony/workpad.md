## Plan

1. Delete the legacy Python CLI shims that only forward to Go:
   - `scripts/ops/bigclaw_github_sync.py`
   - `scripts/ops/bigclaw_refill_queue.py`
2. Update repo-facing migration documentation that still presents those Python entrypoints as active commands so the documented operator path is Go-only.
3. Add a focused Go regression test that asserts:
   - the deleted Python shim files are absent
   - the Go-owned replacement surfaces still exist
4. Run targeted validation for the touched Go regression package and measure the repository Python file count delta.
5. Commit with a message that explicitly lists deleted Python files and added Go test files, then push the branch.

## Acceptance

- Repository Python file count decreases from the pre-change baseline of `60`.
- `scripts/ops/bigclaw_github_sync.py` is deleted.
- `scripts/ops/bigclaw_refill_queue.py` is deleted.
- Documentation no longer instructs operators to use those deleted Python shims as the active path.
- A Go regression test covers the deletion contract and the Go replacement files for the removed shims.
- Targeted tests pass.
- Changes are committed and pushed on the working branch.

## Validation

- `rg --files -g '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche7'`
- `git status --short`
- `git log -1 --stat`
