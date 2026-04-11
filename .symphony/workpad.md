# BIG-GO-257 Workpad

## Plan

1. Confirm the repository-wide `.py` inventory is still zero and verify the
   broad repo directories assigned to `BIG-GO-257` remain Python-free in this
   checkout:
   - `.github`
   - `.githooks`
   - `docs`
   - `reports`
   - `scripts`
   - `bigclaw-go/examples`
   - `bigclaw-go/docs/reports`
2. Add issue-scoped evidence and regression coverage for the broad repo sweep:
   - `bigclaw-go/internal/regression/big_go_257_zero_python_guard_test.go`
   - `bigclaw-go/docs/reports/big-go-257-python-asset-sweep.md`
   - `reports/BIG-GO-257-validation.md`
   - `reports/BIG-GO-257-status.json`
3. Run the targeted inventory checks and the new regression test, then commit
   and push the issue-scoped change set to the remote branch for `BIG-GO-257`.

## Acceptance

- `BIG-GO-257` records a repository-wide zero-Python baseline for the assigned
  broad repo directories without changing unrelated issue evidence.
- The new regression guard fails if any physical `.py` files reappear anywhere
  in the repo or in the assigned broad-sweep directories.
- The sweep report and issue validation artifact capture the retained Go/native
  replacement surface, exact validation commands, and observed results.
- The resulting work is committed and pushed with local and remote SHA parity.

## Validation

- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-257 -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-257/.github /Users/openagi/code/bigclaw-workspaces/BIG-GO-257/.githooks /Users/openagi/code/bigclaw-workspaces/BIG-GO-257/docs /Users/openagi/code/bigclaw-workspaces/BIG-GO-257/reports /Users/openagi/code/bigclaw-workspaces/BIG-GO-257/scripts /Users/openagi/code/bigclaw-workspaces/BIG-GO-257/bigclaw-go/examples /Users/openagi/code/bigclaw-workspaces/BIG-GO-257/bigclaw-go/docs/reports -type f -name '*.py' 2>/dev/null | sort`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-257/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO257(RepositoryHasNoPythonFiles|BroadRepoDirectoriesStayPythonFree|GoNativeSurfaceRemainsAvailable|LaneReportCapturesSweepState)$'`

## Execution Notes

- 2026-04-12: Initial inspection shows the checkout already has a repository-wide
  physical Python file count of `0`.
- 2026-04-12: `BIG-GO-257` therefore lands as issue-scoped regression hardening
  and evidence capture for a broad repo directory sweep rather than in-branch
  Python file deletion.
