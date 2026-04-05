# BIG-GO-1352 Workpad

## Plan

1. Reconfirm the repository-wide physical Python inventory and explicitly verify the `tests` lane focus plus the adjacent residual directories `src/bigclaw`, `scripts`, and `bigclaw-go/scripts`.
2. Add the lane-scoped artifacts for `BIG-GO-1352` so this unattended run records the `tests/*.py` redundancy-removal baseline and the Go/native replacement paths that cover test and operational flows:
   - `bigclaw-go/docs/reports/big-go-1352-tests-python-sweep.md`
   - `reports/BIG-GO-1352-status.json`
   - `reports/BIG-GO-1352-validation.md`
   - `bigclaw-go/internal/regression/big_go_1352_zero_python_guard_test.go`
3. Run targeted validation, record exact commands and results, then commit and push the lane update.

## Acceptance

- The remaining repository Python inventory is explicit, with `tests/*.py` called out as the issue focus.
- The lane either removes redundant `tests/*.py` assets or, when the checkout is already Python-free, lands a concrete Go/native replacement in git that hardens that state.
- Go/native replacement paths for the retired Python test/ops surface are documented in the lane artifacts.
- Exact validation commands and results are recorded.
- The change is committed and pushed to the remote branch.

## Validation

- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1352 -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1352/tests /Users/openagi/code/bigclaw-workspaces/BIG-GO-1352/src/bigclaw /Users/openagi/code/bigclaw-workspaces/BIG-GO-1352/scripts /Users/openagi/code/bigclaw-workspaces/BIG-GO-1352/bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1352/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1352(RepositoryHasNoPythonFiles|TestsDirectoryStaysPythonFree|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
