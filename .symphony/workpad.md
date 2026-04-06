# BIG-GO-1503

## Plan
1. Repair the local checkout so repository contents are available from `origin`.
2. Record the current `.py` file baseline in the repo and identify remaining callers that still depend on physical Python scripts.
3. Switch the remaining callers to the Go-side replacement or another existing non-Python path, keeping changes scoped to that migration.
4. Delete obsolete Python files that are no longer referenced.
5. Run targeted validation for the touched paths, then capture before/after Python file counts and deleted file list.
6. Commit the change and push the branch to the remote.

## Acceptance
- The actual repository `.py` file count decreases versus the pre-change baseline, or the lane records a repository-grounded blocker if the baseline is already zero.
- Remaining callers covered by this issue no longer require deleted Python files, or the lane documents that no Python callers remain.
- Deleted files are truly unused within the repo after the change, or the deleted file inventory is explicitly `none`.
- Validation demonstrates the updated caller path works or the relevant tests pass.
- A commit is created and pushed to the remote branch for this workspace.

## Validation
- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1503(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
- `git status --short`
- `git log --oneline -n 1`
