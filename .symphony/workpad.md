## Plan

1. Delete the Python repo-linkage tranche while preserving the remaining Python closeout consumer in `observability.py`:
   - `src/bigclaw/repo_links.py`
   - `src/bigclaw/repo_plane.py`
   - `tests/test_repo_links.py`
2. Move the minimal `RunCommitLink` / `RunCommitBinding` / role-validation helpers into `src/bigclaw/observability.py` so existing closeout serialization continues to work after the file deletions.
3. Update the one remaining Python test consumer to import `RunCommitLink` from `bigclaw.observability`.
4. Add a focused Go regression test that asserts:
   - the deleted Python files are absent
   - the Go-owned replacement surfaces still exist
5. Run targeted validation for the touched Python syntax and Go regression package, then measure the repository Python file count delta from the current baseline.
6. Commit with a message that explicitly lists deleted Python files and added Go files/Go tests, then push the branch.

## Acceptance

- Repository Python file count decreases from the continuation baseline of `38`.
- `src/bigclaw/repo_links.py` is deleted.
- `src/bigclaw/repo_plane.py` is deleted.
- `tests/test_repo_links.py` is deleted.
- `src/bigclaw/observability.py` still supports run-closeout commit-link serialization without importing deleted Python files.
- A Go regression test covers the deletion contract and the Go replacement files for the removed repo-linkage surface.
- Targeted tests pass.
- Changes are committed and pushed on the working branch.

## Validation

- `python3 -m py_compile src/bigclaw/observability.py tests/test_observability.py`
- `rg --files -g '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche17'`
- `git status --short`
- `git log -1 --stat`
