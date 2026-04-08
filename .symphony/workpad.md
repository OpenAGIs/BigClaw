# BIG-GO-1583 Workpad

## Context
- Issue: `BIG-GO-1583`
- Goal: close the strict `tests/*.py` bucket-A lane with issue-scoped no-regression coverage and publication evidence.
- Current repo state on entry: repository-wide physical Python inventory is already `0`, and the retired root `tests` surface is already absent.

## Scope
- `.symphony/workpad.md`
- `bigclaw-go/internal/regression/big_go_1583_zero_python_guard_test.go`
- `bigclaw-go/docs/reports/big-go-1583-tests-bucket-a-sweep.md`
- `reports/BIG-GO-1583-validation.md`
- `reports/BIG-GO-1583-status.json`

## Plan
1. Replace the stale workpad with an issue-specific plan, acceptance criteria, and validation targets before any code edits.
2. Add a focused regression guard that proves the repository remains Python-free, the retired root `tests` directory stays absent, and the bucket-A `tests/*.py` paths remain absent while their Go/native replacements remain available.
3. Add lane documentation and publication artifacts that record the audited bucket-A paths, replacement evidence, and exact validation commands/results for this lane.
4. Run the targeted inventory and regression commands, record the exact results, then commit and push the issue branch to the remote branch for `BIG-GO-1583`.

## Acceptance
- `BIG-GO-1583` has an issue-specific workpad, regression guard, lane report, validation report, and status artifact.
- The regression guard verifies repository-wide physical Python count remains `0`, the retired root `tests` directory remains absent, and the bucket-A `tests/*.py` paths stay absent.
- The lane report and validation report capture the exact commands and exact results used for repository-wide Python inventory, focused bucket-A path verification, and the targeted Go regression run.
- Changes remain scoped to `BIG-GO-1583` artifacts only.

## Validation
- `find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`
- `find tests -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1583(RepositoryHasNoPythonFiles|RootTestsDirectoryStaysAbsent|BucketATestPathsStayAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesBucketASweep)$'`
