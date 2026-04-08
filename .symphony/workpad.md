# BIG-GO-1589 Workpad

## Plan

1. Confirm the current root `scripts` bucket inventory and identify the Go or
   shell-native replacement entrypoints that own the retired Python surface.
2. Add a lane-specific regression guard for `BIG-GO-1589` that keeps the
   repository Python-free and enforces the zero-`*.py` state under `scripts`.
3. Record lane validation in `reports/BIG-GO-1589-validation.md`, including
   exact commands and results, then commit and push the branch.

## Acceptance

- Root `scripts` physical `.py` count is `0` and stays `0`.
- Lane-specific regression coverage exists for the root `scripts` bucket.
- `reports/BIG-GO-1589-validation.md` records the issue scope, replacement
  surfaces, exact validation commands, and exact results.
- The resulting change is committed and pushed to `origin/BIG-GO-1589`.

## Validation

- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1589 -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1589/scripts -type f -name '*.py' 2>/dev/null | sort`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1589/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1589(RepositoryHasNoPythonFiles|RootScriptsBucketStaysPythonFree|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
