## Plan

1. Delete the remaining isolated Python workspace implementation modules that have Go-owned replacements:
   - `src/bigclaw/workspace_bootstrap.py`
   - `src/bigclaw/workspace_bootstrap_cli.py`
2. Update migration and cutover docs so those removed Python modules are recorded as retired and the active bootstrap/validation path stays on the Go CLI.
3. Add a focused Go regression test that asserts:
   - the deleted Python workspace modules are absent
   - the Go-owned replacement surfaces still exist
4. Run targeted validation for the touched Go regression package and measure the repository Python file count delta from the current baseline.
5. Commit with a message that explicitly lists deleted Python files and added Go test files, then push the branch.

## Acceptance

- Repository Python file count decreases from the continuation baseline of `55`.
- `src/bigclaw/workspace_bootstrap.py` is deleted.
- `src/bigclaw/workspace_bootstrap_cli.py` is deleted.
- Documentation no longer presents those deleted Python modules as active ownership surfaces.
- A Go regression test covers the deletion contract and the Go replacement files for the removed modules.
- Targeted tests pass.
- Changes are committed and pushed on the working branch.

## Validation

- `rg --files -g '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche9'`
- `git status --short`
- `git log -1 --stat`
