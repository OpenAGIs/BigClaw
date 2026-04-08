# BIG-GO-17

## Context
- Issue: `BIG-GO-17`
- Title: `Sweep auxiliary nested Python modules batch B`
- Goal: harden the Go-only repository state for nested auxiliary report and evidence directories outside the primary source and script paths.
- Current repo state on entry: the checked-out workspace already has a repository-wide physical Python file count of `0`, so this lane is expected to add targeted regression coverage and auditable evidence rather than delete live `.py` assets.

## Scope
- `.symphony/workpad.md`
- `bigclaw-go/internal/regression/big_go_17_auxiliary_nested_python_sweep_b_test.go`
- `bigclaw-go/docs/reports/big-go-17-auxiliary-nested-python-sweep-b.md`
- `reports/BIG-GO-17-closeout.md`
- `reports/BIG-GO-17-validation.md`
- `reports/BIG-GO-17-status.json`

## Plan
1. Replace the stale carried-over workpad with issue-specific context, acceptance criteria, and validation commands before editing tracked files.
2. Add a `BIG-GO-17` regression guard that verifies the selected nested auxiliary directories remain Python-free and that representative retained evidence assets still exist.
3. Add a lane report plus status and validation artifacts that document the zero-Python inventory and exact validation results for this sweep.
4. Record repo-native closeout state for the missing local tracker entry, then commit and push the lane.

## Acceptance
- `.symphony/workpad.md` is specific to `BIG-GO-17`.
- `bigclaw-go/internal/regression/big_go_17_auxiliary_nested_python_sweep_b_test.go` fails if Python files reappear under the selected nested auxiliary directories.
- The lane report documents the scoped directories, representative retained native assets, and the exact validation commands for `BIG-GO-17`.
- The closeout note explains that no writable `BIG-GO-17` tracker record exists in this workspace.
- `reports/BIG-GO-17-validation.md` and `reports/BIG-GO-17-status.json` record exact commands and exact results from this lane.
- Changes remain scoped to `BIG-GO-17`.

## Validation
- `find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`
- `find docs/reports bigclaw-go/docs/reports/broker-failover-stub-artifacts bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts bigclaw-go/docs/reports/live-shadow-runs bigclaw-go/docs/reports/live-validation-runs -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO17(NestedAuxiliaryDirectoriesStayPythonFree|RetainedNativeEvidenceAssetsRemainAvailable|LaneReportCapturesSweepState)$'`
