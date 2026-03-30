# BIG-GO-987

## Plan
- Inventory Python files related to repo, governance, and reporting tests plus corresponding Go implementations/tests.
- Determine which Python tests are fully superseded by Go coverage and can be removed without expanding scope.
- Apply scoped deletions or minimal follow-up edits needed to keep the repository consistent.
- Run targeted validation for affected Go packages and repository Python-file counts.
- Commit the change set and push the branch to the remote.

## Acceptance
- Produce an explicit list of Python files in this batch.
- Reduce Python file count in the targeted repo/governance/reporting test area where replacement coverage already exists.
- Record the rationale for each retained, replaced, or deleted file.
- Report the net effect on total repository Python file count.

## Validation
- Use `rg --files -g '*.py'` before and after changes to measure total Python file count.
- Run targeted `go test` commands for replacement coverage in repo, governance, and reporting packages.
- Confirm git status only includes issue-scoped edits before committing.
