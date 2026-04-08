# BIG-GO-119 Workpad

## Context
- Issue: `BIG-GO-119`
- Goal: harden the zero-Python repository baseline by sweeping residual auxiliary surfaces in nested, hidden, and lower-priority directories.
- Current repo state on entry: repository-wide physical `.py` inventory is already `0`, including the main priority directories `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.

## Scope
- `.symphony/workpad.md`
- `bigclaw-go/internal/regression/big_go_119_zero_python_guard_test.go`
- `bigclaw-go/docs/reports/big-go-119-python-asset-sweep.md`
- `reports/BIG-GO-119-status.json`
- `reports/BIG-GO-119-validation.md`

## Plan
1. Replace the stale workpad with issue-specific scope, acceptance, and validation targets before code edits.
2. Add a lane-specific regression guard for repository-wide zero Python, priority residual directories, and the hidden or lower-priority directories audited by this lane.
3. Add lane reports that capture the zero-Python inventory, hidden-directory sweep coverage, and active Go or shell-native replacement surfaces.
4. Run targeted inventory and regression commands, record exact commands and exact results, then commit and push the lane branch to `origin/main`.

## Acceptance
- `BIG-GO-119` has an issue-specific workpad, regression guard, lane report, status artifact, and validation report.
- The regression guard verifies repository-wide Python file count `0`, keeps priority directories Python-free, and locks down the hidden or lower-priority sweep directories audited in this lane.
- Validation records exact commands and exact results for repository inventory, hidden-directory inventory, and targeted regression coverage.
- Changes remain scoped to `BIG-GO-119` artifacts only.

## Validation
- `find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`
- `find .github .githooks .symphony docs bigclaw-go/docs bigclaw-go/examples scripts/ops -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO119(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|HiddenAndAuxiliaryDirectoriesStayPythonFree|ReplacementPathsRemainAvailable|LaneReportDocumentsPythonAssetSweep)$'`
