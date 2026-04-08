# BIG-GO-164

## Context
- Issue: `BIG-GO-164`
- Goal: harden the residual shell wrapper and CLI-helper surface so the repo stays Go-only for script entrypoints after the Python shim removals already landed.
- Current repo state on entry: the checkout already has `0` physical `.py` files, so this lane is a scoped regression-prevention sweep rather than a fresh file-deletion batch.

## Scope
- `.symphony/workpad.md`
- `README.md`
- `bigclaw-go/internal/regression/big_go_164_zero_python_guard_test.go`
- `bigclaw-go/docs/reports/big-go-164-python-asset-sweep.md`
- `reports/BIG-GO-164-validation.md`
- `reports/BIG-GO-164-status.json`

## Plan
1. Replace the stale carried-over workpad with issue-specific plan, acceptance, and validation before editing tracked files.
2. Add a focused regression guard for the residual script-helper surface: root shell wrappers, root bootstrap helper, and retained benchmark/e2e shell entrypoints.
3. Refresh README guidance only where needed so the retained helper paths are explicitly documented as Go/shell-only replacements for the retired Python wrappers.
4. Record exact validation commands and exact results in the lane report and validation/status artifacts.
5. Commit the scoped change set and push the issue branch to the remote.

## Acceptance
- The workpad is specific to `BIG-GO-164`.
- The residual shell helper surface remains Python-free and covered by a focused Go regression guard.
- The lane report records repository-wide and helper-surface Python inventories as `0`.
- README guidance explicitly matches the retained Go/shell helper paths for the residual script-helper surface.
- Validation artifacts record the exact commands run and their exact results.
- Changes remain scoped to this issue.

## Validation
- `find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`
- `find scripts scripts/ops bigclaw-go/scripts/benchmark bigclaw-go/scripts/e2e -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO164(RepositoryHasNoPythonFiles|ResidualScriptHelperSurfaceStaysPythonFree|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
