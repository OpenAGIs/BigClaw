# BIG-GO-140 Workpad

## Context
- Issue: `BIG-GO-140`
- Title: `Convergence sweep toward <=1 Python file I`
- Goal: execute a scoped convergence sweep that documents the repository's current practical Go-only state and hardens it with lane-specific regression coverage.
- Current repo state on entry: the checked-out `origin/main` baseline already has no physical `*.py` files anywhere in the repository, so this lane can only strengthen evidence and guardrails rather than delete an in-branch Python file.

## Scope
- `.symphony/workpad.md`
- `bigclaw-go/internal/regression/big_go_140_zero_python_guard_test.go`
- `bigclaw-go/docs/reports/big-go-140-python-asset-sweep.md`
- `reports/BIG-GO-140-status.json`
- `reports/BIG-GO-140-validation.md`

## Plan
1. Replace the stale workpad with this issue-specific plan, acceptance criteria, and validation targets before editing any lane artifacts.
2. Add a `BIG-GO-140` regression guard that asserts the repository and priority residual directories remain free of physical Python files and that the current Go/native replacement paths still exist.
3. Add the lane report and status/validation artifacts that record the zero-Python baseline, exact validation commands, exact results, and the lane-specific blocker that no `.py` file remained to remove in this checkout.
4. Run the targeted Python-inventory commands and the focused regression test command, capture the exact outputs, then commit and push the scoped branch changes.

## Acceptance
- `bigclaw-go/internal/regression/big_go_140_zero_python_guard_test.go` exists and enforces repository-wide zero physical `*.py` files, Python-free priority directories, replacement-path availability, and lane-report content.
- `bigclaw-go/docs/reports/big-go-140-python-asset-sweep.md` records the repository-wide and priority-directory Python inventory as zero and includes exact validation commands plus results.
- `reports/BIG-GO-140-status.json` and `reports/BIG-GO-140-validation.md` reflect the lane title, artifacts, validation commands, exact results, and the zero-baseline blocker.
- The change stays scoped to `BIG-GO-140` convergence sweep artifacts only.

## Validation
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-140 -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-140/src/bigclaw /Users/openagi/code/bigclaw-workspaces/BIG-GO-140/tests /Users/openagi/code/bigclaw-workspaces/BIG-GO-140/scripts /Users/openagi/code/bigclaw-workspaces/BIG-GO-140/bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-140/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO140(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
