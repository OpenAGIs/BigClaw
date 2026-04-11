# BIG-GO-248 Workpad

## Plan

1. Confirm the repository-wide Python inventory and inspect adjacent zero-Python
   sweep tickets to identify the expected `BIG-GO-248` artifact pattern and
   evidence scope.
2. Add the issue-scoped `BIG-GO-248` evidence bundle for this broad repo Python
   reduction sweep:
   - `bigclaw-go/internal/regression/big_go_248_zero_python_guard_test.go`
   - `bigclaw-go/docs/reports/big-go-248-python-asset-sweep.md`
   - `reports/BIG-GO-248-validation.md`
   - `reports/BIG-GO-248-status.json`
3. Run the targeted repository inventory checks and the `BIG-GO-248`
   regression validation, then commit and push the resulting branch state.

## Acceptance

- `BIG-GO-248` records a repo-visible, issue-scoped proof that the checkout
  remains free of tracked `.py` files.
- The new regression guard verifies the repository-wide zero-Python baseline,
  the selected priority residual directories for this sweep, the retained
  replacement paths, and the `BIG-GO-248` lane report.
- The validation and status artifacts capture the exact commands and observed
  results for this issue, including the already-zero baseline caveat.
- The final branch state is committed and pushed to `origin`.

## Validation

- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-248 -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-248/docs /Users/openagi/code/bigclaw-workspaces/BIG-GO-248/reports /Users/openagi/code/bigclaw-workspaces/BIG-GO-248/bigclaw-go/docs/reports /Users/openagi/code/bigclaw-workspaces/BIG-GO-248/bigclaw-go/internal/migration -type f -name '*.py' 2>/dev/null | sort`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-248/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO248(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
- `jq '.' /Users/openagi/code/bigclaw-workspaces/BIG-GO-248/reports/BIG-GO-248-status.json >/dev/null`

## Execution Notes

- 2026-04-12: Initial inspection shows the checkout is already at a
  repository-wide Python file count of `0`.
- 2026-04-12: `BIG-GO-248` therefore needs to harden the zero-Python baseline
  with issue-scoped regression and reporting artifacts instead of removing
  in-branch `.py` files.
