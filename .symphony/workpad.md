# BIG-GO-1514 Workpad

## Plan

1. Reconfirm the repository-wide physical Python file inventory, with explicit focus on `scripts` and `scripts/ops`.
2. Inspect the current `scripts` and `scripts/ops` tree plus lane-relevant history to identify any remaining refill-wrapper references or stale deletion gaps.
3. Apply only the lane-scoped cleanup and regression coverage needed to keep the removed Python wrapper paths absent, then capture before/after evidence.
4. Run targeted validation, record exact commands and results, commit the change, and push the issue branch.

## Acceptance

- The lane stays scoped to refill-related cleanup under `scripts` and `scripts/ops`.
- Before/after repository `.py` counts are recorded explicitly.
- Deleted-file evidence is recorded for the retired Python refill wrapper path.
- Exact validation commands and outcomes are recorded.
- The branch is committed and pushed to the remote.

## Validation

- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1514 -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1514/scripts /Users/openagi/code/bigclaw-workspaces/BIG-GO-1514/scripts/ops -type f -name '*.py' -print | sort`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1514/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1514(ScriptsDirectoriesRemainPythonFree|RetiredRefillWrapperStaysDeleted|RefillWrapperDeletionEvidenceIsRecorded)$'`
