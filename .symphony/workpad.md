# BIG-GO-1543

## Plan
- Audit the current `bigclaw-go/scripts` Python inventory and capture exact before-state counts and file lists.
- If any `bigclaw-go/scripts/*.py` files remain, delete only those files and record the exact removed-file list.
- If the baseline is already zero-Python for `bigclaw-go/scripts`, land issue-scoped validation artifacts and a regression guard that preserve the exact `0 -> 0` sweep state.
- Run targeted validation commands, record exact commands and results, then commit and push the issue branch.

## Acceptance
- `bigclaw-go/scripts` contains no physical `.py` files after the lane work.
- Before and after counts for `bigclaw-go/scripts/*.py` are recorded exactly.
- The exact removed-file list is recorded exactly, including an explicit empty list when no files remained at baseline.
- Validation commands and results are captured in issue-scoped artifacts.
- Changes remain scoped to `BIG-GO-1543`.
- The lane branch is committed and pushed to `origin`.

## Validation
- `find bigclaw-go/scripts -type f -name '*.py' | sort`
- `find bigclaw-go/scripts -type f -name '*.py' | wc -l`
- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1543(BigClawGoScriptsStayPythonFree|GoReplacementPathsRemainAvailable|LaneReportCapturesExactSweepState)$'`
- `git diff --name-status`
- `git status --short`
