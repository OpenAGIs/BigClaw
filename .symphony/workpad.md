## Plan

1. Purge the first safe top-level Python tranche under `src/bigclaw` by deleting:
   - `src/bigclaw/cost_control.py`
   - `src/bigclaw/issue_archive.py`
   - `src/bigclaw/github_sync.py`
2. Remove any package exports that still point at those deleted Python modules so `src/bigclaw/__init__.py` no longer imports them.
3. Add a focused Go regression test that asserts the migration contract for this tranche:
   - the deleted Python files are absent
   - the corresponding Go replacement files exist
4. Run targeted validation for the touched Go packages and the new regression test.
5. Commit with a message that explicitly lists deleted Python files and added Go test files, then push the branch.

## Acceptance

- Python file count in the repository decreases from the pre-change baseline.
- `src/bigclaw/cost_control.py`, `src/bigclaw/issue_archive.py`, and `src/bigclaw/github_sync.py` are deleted.
- `src/bigclaw/__init__.py` no longer imports symbols from deleted modules.
- A Go test covers the tranche replacement contract against the repository tree.
- Targeted Go tests pass.
- Changes are committed and pushed to the remote branch for `BIG-GO-1041`.

## Validation

- `find . -name '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/costcontrol ./internal/issuearchive ./internal/githubsync ./internal/regression -run 'TestTopLevelModulePurgeTranche1'`
- `git status --short`
- `git log -1 --stat`
