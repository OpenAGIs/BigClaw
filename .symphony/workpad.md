# BIG-GO-139 Workpad

## Context
- Issue: `BIG-GO-139`
- Goal: harden the zero-Python repository baseline by sweeping hidden, nested, and overlooked report-heavy auxiliary directories.
- Current repo state on entry: repository-wide physical Python inventory is already `0`, including all priority removal directories.

## Scope
- `.symphony/workpad.md`
- `bigclaw-go/internal/regression/big_go_139_zero_python_guard_test.go`
- `bigclaw-go/docs/reports/big-go-139-python-asset-sweep.md`
- `reports/BIG-GO-139-status.json`
- `reports/BIG-GO-139-validation.md`

## Plan
1. Replace the stale workpad with an issue-specific plan, acceptance criteria, and validation targets before any code edits.
2. Add a lane-specific regression guard for repository-wide zero Python, the standard priority directories, and the overlooked report-heavy auxiliary directories covered by this lane.
3. Add lane artifacts that document the audited directories, the retained non-Python report surfaces, and the exact validation commands for this sweep.
4. Run targeted inventory and regression commands, record exact commands and results, then commit and push the issue branch state to `origin/main`.

## Acceptance
- `BIG-GO-139` has an issue-specific workpad, regression guard, lane report, validation report, and status artifact.
- The regression guard verifies repository-wide Python file count `0`, keeps the priority directories Python-free, and locks down the report-heavy auxiliary directories audited in this lane.
- The lane report and validation report record exact commands and exact results for the repository inventory, the lane-specific auxiliary directory inventory, and the targeted Go regression run.
- Changes remain scoped to `BIG-GO-139` artifacts only.

## Validation
- `find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`
- `find reports docs/reports bigclaw-go/docs/reports bigclaw-go/docs/reports/live-shadow-runs bigclaw-go/docs/reports/live-validation-runs -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO139(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|ReportHeavyAuxiliaryDirectoriesStayPythonFree|RetainedNativeReportAssetsRemainAvailable|LaneReportDocumentsPythonAssetSweep)$'`
