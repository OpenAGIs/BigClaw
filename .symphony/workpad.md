# BIG-GO-1510 Workpad

## Plan

1. Reconfirm the repository-wide physical Python asset inventory, with explicit
   checks for `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.
2. Add BIG-GO-1510-scoped reporting and regression coverage that records the
   live repo-reality counts, the explicit deleted-file list, and the active
   Go/native replacement paths.
3. Run targeted validation, capture exact commands and results in the issue
   artifacts, then commit and push `BIG-GO-1510`.

## Acceptance

- `.symphony/workpad.md` exists before any non-workpad file changes.
- The lane records before/after physical `.py` file counts from the live
  checkout and states the deleted file list explicitly.
- The lane keeps changes scoped to BIG-GO-1510 artifacts and regression
  coverage.
- Exact validation commands and outcomes are recorded.
- The branch is committed and pushed to `origin/BIG-GO-1510`.

## Validation

- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1510 -path '*/.git' -prune -o -type f -name '*.py' -print | sort`
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1510 -path '*/.git' -prune -o -type f -name '*.py' -print | wc -l`
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1510/src/bigclaw /Users/openagi/code/bigclaw-workspaces/BIG-GO-1510/tests /Users/openagi/code/bigclaw-workspaces/BIG-GO-1510/scripts /Users/openagi/code/bigclaw-workspaces/BIG-GO-1510/bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1510/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1510(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|GoReplacementPathsRemainAvailable|LaneArtifactsCaptureZeroPythonReality)$'`
