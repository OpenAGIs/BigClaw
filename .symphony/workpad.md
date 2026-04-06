# BIG-GO-1512

## Plan

1. Use the residual Python branch state from `BIG-GO-1511` as the issue base and capture the starting Python file count.
2. Verify whether any Python test or `conftest.py` blockers still exist on that base.
3. Delete the remaining root ops Python wrappers so the repository-wide `.py` count decreases physically.
4. Add regression coverage and doc updates that keep the deleted wrappers absent and point operators at `bash scripts/ops/bigclawctl ...`.
5. Recount Python files, collect deleted-file evidence, run targeted validation, then commit and push the issue branch.

## Acceptance

- Physical `.py` file count in the repository decreases from the residual issue base.
- The change includes actual deleted Python files, not status-only churn.
- Before and after counts are recorded.
- Targeted tests or validation commands are run and their exact results are captured.
- Changes remain scoped to this issue.
- The branch is committed and pushed to the remote.

## Validation

- `git fetch origin`
- `git fetch ../BIG-GO-1511 BIG-GO-1511:refs/remotes/local1511/BIG-GO-1511`
- `git checkout -B BIG-GO-1512 local1511/BIG-GO-1511`
- `find . -type f -name '*.py' | sort | wc -l`
- `cd bigclaw-go && go test ./internal/regression -run 'Test(PythonTestTranche14Removed|BIGGO1512RootOpsPythonWrappersRemoved|BIGGO1512RootOpsDocsUseGoEntrypoints)$'`
- `bash scripts/ops/bigclawctl refill --help >/dev/null && bash scripts/ops/bigclawctl workspace bootstrap --help >/dev/null && bash scripts/ops/bigclawctl workspace validate --help >/dev/null`
- `git diff --name-status`
- `git status --short`
