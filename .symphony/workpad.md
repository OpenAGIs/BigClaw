# BIG-GO-1170

## Plan
- Materialize the empty BIG-GO-1170 workspace onto the repository mainline so candidate files can be inspected.
- Measure current Python file count and verify whether any candidate files still exist in this workspace.
- If real Python files remain in scope, remove or replace them with the existing Go-compatible path used by the current repository.
- Run targeted validation commands and capture exact commands plus results.
- Commit scoped changes and push the issue branch.

## Acceptance
- Cover real Python files that still exist in this workspace and are in scope for this issue.
- Verify the Go replacement or compatible non-Python path for any removed Python entrypoint.
- Reduce the actual result of `find . -name '*.py' | wc -l` in this workspace.

## Validation
- `find . -name '*.py' | wc -l`
- Repository-specific checks for any touched benchmark/e2e/migration scripts.
- `git status --short`
