## Plan

1. Inspect the current repo and CI/test entrypoints that still allow Python-first validation paths.
2. Add a Go-native regression/enforcement test that fails when new repository Python files appear outside an explicit frozen allowlist.
3. Remove `scripts/ops/symphony_workspace_validate.py`, an existing Python wrapper whose active replacement is `bigclawctl workspace validate`, so the repository `.py` count decreases in the same change.
4. Wire the new enforcement into the root validation path so `make test` and CI exercise the Go-native Python-ban guard.
5. Run targeted validation for the touched Go packages and record exact commands and outcomes.
6. Commit with a message that explicitly lists deleted Python files and added Go files/Go tests, then push the branch.

## Acceptance

- Repository Python file count decreases from the pre-change baseline.
- A repo-wide Go regression guard exists to block newly added Python files outside the frozen allowlist.
- The guard runs from standard repository validation entrypoints rather than being dead code.
- `scripts/ops/symphony_workspace_validate.py` is deleted and the Go workspace-validate entrypoint remains present.
- Targeted Go validation passes for the touched paths.
- The commit message lists the deleted Python file(s) and the added Go file(s)/Go test file(s).
- Changes are committed and pushed to the remote branch for `BIG-GO-1049`.

## Validation

- `find . -name '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/regression -run 'TestRepoWidePythonFileBan|TestWorkspaceValidatePythonShimRemoved'`
- `make test`
- `git status --short`
- `git log -1 --stat`
